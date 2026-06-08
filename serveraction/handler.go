package serveraction

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/polagonow/pola/core"
)

// maxBodyBytes caps the request body size for a server-action call.
const maxBodyBytes = 8 << 20 // 8 MiB

// Handler serves the server-action HTTP endpoints. It is constructed once at
// wiring time and its Action/Form methods are registered on the orchestrator.
type Handler struct {
	// Invoker executes a resolved action in the renderer's JS runtime pool.
	Invoker core.ServerActionInvoker
	// ValidIDs is the set of known action module IDs; calls referencing an
	// unknown id are rejected before any JS runs.
	ValidIDs map[string]struct{}
	// BuildContext derives the per-request __REQUEST__ context (headers, url,
	// method) for an action call. Supplied by the pipeline (reuses the
	// orchestrator's request-context builder).
	BuildContext func(*http.Request) map[string]any
	// Injectors expose Go services to the action's runtime. These are the same
	// per-request-memoized injectors used for page renders.
	Injectors []core.RuntimeInjector
	// Dev selects relaxed argument-validation limits when true.
	Dev bool
	// Logger records handler-level errors. May be nil.
	Logger core.Logger
	// Invalidate, when non-nil, is called with an affected path after a
	// successful mutating action so the SSR cache reflects the new state.
	Invalidate func(ctx context.Context, path string)
}

// actionRequest is the JSON body of a POST /_pola/action call. The field names
type actionRequest struct {
	ID         string `json:"id"`
	ExportName string `json:"export_name"`
	Args       []any  `json:"args"`
}

// Action handles POST /_pola/action (JSON-RPC style server-action calls).
func (h *Handler) Action(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	if r.Method != http.MethodPost {
		writeFail(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if err := CheckOrigin(r); err != nil {
		writeFail(w, http.StatusForbidden, "Forbidden")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		writeFail(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	var req actionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeFail(w, http.StatusBadRequest, "Invalid request format")
		return
	}
	if len(req.Args) > MaxBoundArgs {
		writeFail(w, http.StatusBadRequest, "Server action has too many bound arguments")
		return
	}
	if IsReservedExportName(req.ExportName) {
		writeFail(w, http.StatusBadRequest, "Invalid export name: reserved for internal use")
		return
	}
	if _, ok := h.ValidIDs[req.ID]; !ok {
		writeFail(w, http.StatusNotFound, "Unknown server action")
		return
	}

	sanitized, err := ValidateAndSanitizeArgs(req.Args, ConfigFor(h.Dev))
	if err != nil {
		writeFail(w, http.StatusBadRequest, "Input validation failed: "+err.Error())
		return
	}

	out, err := h.Invoker.InvokeAction(r.Context(), core.InvokeInput{
		ID:             req.ID,
		ExportName:     req.ExportName,
		Args:           sanitized,
		RequestContext: h.buildContext(r),
		Injectors:      h.Injectors,
	})
	if err != nil {
		h.logError("pola: server action execution", "id", req.ID, "export", req.ExportName, "err", err)
		// Execution errors are returned as a 200 envelope (rari semantics).
		writeEnvelope(w, http.StatusOK, core.InvokeOutput{Success: false, Error: err.Error()})
		return
	}

	if out.Redirect != "" {
		h.invalidate(r.Context(), out.Redirect)
	}
	writeEnvelope(w, http.StatusOK, out)
}

// Form handles POST /_pola/form-action (HTML <form> submissions). It follows the
// Post/Redirect/Get pattern: on success it issues a 303 redirect.
func (h *Handler) Form(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := CheckOrigin(r); err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id := r.PostForm.Get("__action_id")
	exportName := r.PostForm.Get("__export_name")
	if id == "" || exportName == "" {
		http.Error(w, "Missing action metadata", http.StatusBadRequest)
		return
	}
	if IsReservedExportName(exportName) {
		http.Error(w, "Invalid export name", http.StatusBadRequest)
		return
	}
	if _, ok := h.ValidIDs[id]; !ok {
		http.Error(w, "Unknown server action", http.StatusNotFound)
		return
	}

	args := formToArgs(r.PostForm)
	sanitized, err := ValidateAndSanitizeArgs(args, ConfigFor(h.Dev))
	if err != nil {
		http.Error(w, "Input validation failed", http.StatusBadRequest)
		return
	}

	out, err := h.Invoker.InvokeAction(r.Context(), core.InvokeInput{
		ID:             id,
		ExportName:     exportName,
		Args:           sanitized,
		RequestContext: h.buildContext(r),
		Injectors:      h.Injectors,
	})
	if err != nil {
		h.logError("pola: form action execution", "id", id, "export", exportName, "err", err)
		http.Error(w, "Server action failed", http.StatusInternalServerError)
		return
	}

	// Determine the redirect target: an explicit redirect wins; otherwise fall
	// back to the referring page (PRG).
	target := out.Redirect
	if target == "" {
		target = refererPath(r)
	}
	h.invalidate(r.Context(), target)
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusSeeOther)
}

// formToArgs converts URL-encoded form fields into action arguments. Fields
// prefixed with "__" are metadata and skipped. the action receives (prevState=null, formData).
func formToArgs(form url.Values) []any {
	data := make(map[string]any, len(form))
	for key, vals := range form {
		if strings.HasPrefix(key, "__") {
			continue
		}
		if len(vals) > 0 {
			data[key] = vals[0]
		}
	}
	return []any{nil, data}
}

func refererPath(r *http.Request) string {
	ref := r.Header.Get("Referer")
	if ref == "" {
		return "/"
	}
	if u, err := url.Parse(ref); err == nil {
		if u.RawQuery != "" {
			return u.Path + "?" + u.RawQuery
		}
		if u.Path != "" {
			return u.Path
		}
	}
	return "/"
}

func (h *Handler) buildContext(r *http.Request) map[string]any {
	if h.BuildContext == nil {
		return nil
	}
	return h.BuildContext(r)
}

func (h *Handler) invalidate(ctx context.Context, target string) {
	if h.Invalidate == nil || target == "" {
		return
	}
	path := target
	if u, err := url.Parse(target); err == nil && u.Path != "" {
		path = u.Path
	}
	h.Invalidate(ctx, path)
}

func (h *Handler) logError(msg string, args ...any) {
	if h.Logger != nil {
		h.Logger.Error(msg, args...)
	}
}

// envelope is the JSON response shape for /_pola/action
type envelope struct {
	Success  bool            `json:"success"`
	Result   json.RawMessage `json:"result"`
	Error    string          `json:"error,omitempty"`
	Redirect string          `json:"redirect,omitempty"`
}

func writeEnvelope(w http.ResponseWriter, status int, out core.InvokeOutput) {
	result := out.Result
	if len(result) == 0 {
		result = json.RawMessage("null")
	}
	writeJSON(w, status, envelope{
		Success:  out.Success,
		Result:   result,
		Error:    out.Error,
		Redirect: out.Redirect,
	})
}

func writeFail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, envelope{Success: false, Result: json.RawMessage("null"), Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
