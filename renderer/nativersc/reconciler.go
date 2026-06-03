package nativersc

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/polagonow/pola/core"

	gojalib "github.com/dop251/goja"
)

// JS globals the generated server entry installs for the Go-driven walk.
const (
	// createRootFn returns React.createElement(Page, props) for an export.
	createRootFn = "__createRoot__"
	// rscHelper is the classification/execution helper object.
	rscHelper = "__rsc__"
)

// Async resolution tuning. The event loop is driven in short bursts between which
// bridge goroutines (Go services invoked via __DEPENDENCY_INJECTION__) make
// progress; pendingPollInterval gives them time without busy-spinning.
const (
	pendingPollInterval = 1 * time.Millisecond
	pendingTimeout      = 30 * time.Second
)

// reconciler walks a React element tree produced in goja and serializes it to the
// Flight wire format.
//
// All goja access happens on the event loop, but the walk is orchestrated from Go
// across multiple RunRender calls so async server components can settle (the loop
// is not reentrant, so we cannot block-await inside a single run). Pass 1 emits
// the synchronous shell, recording each pending async subtree as a "$L<id>"
// reference; the drain phase then resolves each promise by driving the loop and
// streams its model row.
type reconciler struct {
	lr         loopRenderer
	rt         *gojalib.Runtime  // set at the start of every RunRender callback
	fw         *flightWriter
	rsc        *gojalib.Object   // the __rsc__ helper (resolved on first run)
	importURLs map[string]string // moduleID -> browser chunk URL
	clientRefs map[string]int    // "moduleID#export" -> import row id
	pending    []pendingNode
	logger     core.Logger
}

// pendingNode is an async subtree awaiting resolution. Its resolved value streams
// later as model row id, which the shell references via "$L<id>".
type pendingNode struct {
	id      int
	promise *gojalib.Object
}

func newReconciler(lr loopRenderer, fw *flightWriter, importURLs map[string]string, logger core.Logger) *reconciler {
	return &reconciler{
		lr:         lr,
		fw:         fw,
		importURLs: importURLs,
		clientRefs: map[string]int{},
		logger:     logger,
	}
}

// render executes the page export and writes the full Flight payload: the shell
// model (row 0) plus streamed rows for each async boundary as it resolves.
func (r *reconciler) render(exportName, propsJSON string) error {
	modelID := r.fw.nextID() // reserve row 0 for the root model

	var model any
	err := r.lr.RunRender(func(rt *gojalib.Runtime) error {
		r.rt = rt
		if err := r.ensureHelper(); err != nil {
			return err
		}
		createRoot, ok := gojalib.AssertFunction(rt.Get(createRootFn))
		if !ok {
			return fmt.Errorf("nativersc: %s is not a function", createRootFn)
		}
		root, err := createRoot(gojalib.Undefined(), rt.ToValue(exportName), rt.ToValue(propsJSON))
		if err != nil {
			return fmt.Errorf("nativersc: %s: %w", createRootFn, err)
		}
		model, err = r.walkValue(root)
		return err
	})
	if err != nil {
		return err
	}
	if err := r.fw.writeModel(modelID, model); err != nil {
		return err
	}

	return r.drainPending()
}

// drainPending resolves recorded async subtrees, streaming each resolved model
// row (or an error row). Resolving a subtree may discover further async nodes,
// which are appended and drained in turn.
func (r *reconciler) drainPending() error {
	for len(r.pending) > 0 {
		p := r.pending[0]
		r.pending = r.pending[1:]

		resolved, rerr := r.awaitPromise(p.promise)
		if rerr != nil {
			if err := r.fw.writeError(p.id, "", rerr.Error(), nil); err != nil {
				return err
			}
			continue
		}

		var model any
		err := r.lr.RunRender(func(rt *gojalib.Runtime) error {
			r.rt = rt
			var werr error
			model, werr = r.walkValue(resolved)
			return werr
		})
		if err != nil {
			if werr := r.fw.writeError(p.id, "", err.Error(), nil); werr != nil {
				return werr
			}
			continue
		}
		if err := r.fw.writeModel(p.id, model); err != nil {
			return err
		}
	}
	return nil
}

// ensureHelper resolves the __rsc__ helper object once. Must run on the loop.
func (r *reconciler) ensureHelper() error {
	if r.rsc != nil {
		return nil
	}
	helper := r.rt.Get(rscHelper)
	if helper == nil || gojalib.IsUndefined(helper) {
		return fmt.Errorf("nativersc: %s helper not installed by server entry", rscHelper)
	}
	r.rsc = helper.ToObject(r.rt)
	return nil
}

// awaitPromise drives the event loop until the promise settles, returning its
// resolved value. It runs OUTSIDE any RunRender: it attaches then/catch handlers
// in one run, then drives the loop in short bursts (sequential, never nested)
// until a handler fires.
func (r *reconciler) awaitPromise(promise *gojalib.Object) (gojalib.Value, error) {
	var result gojalib.Value
	var rejErr error
	settled := false

	attachErr := r.lr.RunRender(func(rt *gojalib.Runtime) error {
		r.rt = rt
		thenFn, ok := gojalib.AssertFunction(promise.Get("then"))
		if !ok {
			result = promise // not actually a thenable; treat as the value
			settled = true
			return nil
		}
		onFulfilled := rt.ToValue(func(c gojalib.FunctionCall) gojalib.Value {
			result = c.Argument(0)
			settled = true
			return gojalib.Undefined()
		})
		onRejected := rt.ToValue(func(c gojalib.FunctionCall) gojalib.Value {
			msg := "nativersc: async render rejected"
			if a := c.Argument(0); !gojalib.IsUndefined(a) && !gojalib.IsNull(a) {
				msg = a.String()
			}
			rejErr = fmt.Errorf("%s", msg)
			settled = true
			return gojalib.Undefined()
		})
		_, err := thenFn(promise, onFulfilled, onRejected)
		return err
	})
	if attachErr != nil {
		return nil, attachErr
	}

	deadline := time.Now().Add(pendingTimeout)
	for !settled {
		// An empty run still drains microtasks and processes RunOnLoop jobs that
		// bridge goroutines queued since the last run, advancing the promise.
		if err := r.lr.RunRender(func(rt *gojalib.Runtime) error { r.rt = rt; return nil }); err != nil {
			return nil, err
		}
		if settled {
			break
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("nativersc: async render timed out after %s", pendingTimeout)
		}
		time.Sleep(pendingPollInterval)
	}
	if rejErr != nil {
		return nil, rejErr
	}
	if result == nil {
		result = gojalib.Undefined()
	}
	return result, nil
}

// walkValue converts a single goja value into a Flight model value, executing
// server components and recursing through host elements, fragments, arrays and
// objects. Client references become lazy refs backed by import rows; async
// subtrees become lazy refs whose rows stream later. Runs on the loop.
func (r *reconciler) walkValue(v gojalib.Value) (any, error) {
	kind, err := r.callString("kind", v)
	if err != nil {
		return nil, err
	}

	switch kind {
	case "null", "undefined":
		return nil, nil
	case "string":
		return escapeUserString(v.String()), nil
	case "number":
		return v.Export(), nil
	case "boolean":
		return v.ToBoolean(), nil
	case "array":
		return r.walkArray(v)
	case "object":
		return r.walkObject(v)
	case "host":
		return r.walkHost(v)
	case "component":
		return r.walkComponent(v)
	case "client":
		return r.walkClient(v)
	case "clientref":
		return r.walkClientRef(v)
	case "fragment":
		return r.walkFragment(v)
	case "suspense":
		return r.walkSuspense(v)
	case "promise":
		return r.deferAsync(v)
	case "date":
		// React encodes Date as "$D" + date.toISOString().
		iso, err := r.callString("dateISO", v)
		if err != nil {
			return nil, err
		}
		return "$D" + iso, nil
	case "bigint":
		// React encodes BigInt as "$n" + value.toString().
		return "$n" + v.String(), nil
	case "function":
		// Functions cannot be serialized to the client; drop them.
		return nil, nil
	default:
		// symbol and other exotic values: handled in a later phase. Fall back to
		// the exported value so the render does not abort.
		if r.logger != nil {
			r.logger.Warn("nativersc: unhandled node kind", "kind", kind)
		}
		return v.Export(), nil
	}
}

// walkHost serializes a host element ["$", tag, key, props].
func (r *reconciler) walkHost(v gojalib.Value) (any, error) {
	obj := v.ToObject(r.rt)
	tag := obj.Get("type").String()
	key, err := r.walkValue(obj.Get("key"))
	if err != nil {
		return nil, err
	}
	props, err := r.walkProps(obj.Get("props"))
	if err != nil {
		return nil, err
	}
	return []any{elementMarker{}, tag, key, props}, nil
}

// walkComponent executes a server component function and recurses on its result.
// Async components return a promise, which walkValue defers.
func (r *reconciler) walkComponent(v gojalib.Value) (any, error) {
	result, err := r.call("callComponent", v)
	if err != nil {
		return nil, fmt.Errorf("nativersc: render component: %w", err)
	}
	return r.walkValue(result)
}

// walkClient serializes a client-component reference: an import row plus a lazy
// element tuple ["$", "$L<hex>", key, props].
func (r *reconciler) walkClient(v gojalib.Value) (any, error) {
	obj := v.ToObject(r.rt)
	id, err := r.callString("clientId", v)
	if err != nil {
		return nil, err
	}
	moduleID, export := splitModuleExport(id)
	refID, err := r.clientRefID(moduleID, export)
	if err != nil {
		return nil, err
	}
	key, err := r.walkValue(obj.Get("key"))
	if err != nil {
		return nil, err
	}
	props, err := r.walkProps(obj.Get("props"))
	if err != nil {
		return nil, err
	}
	return []any{elementMarker{}, ref{lazy: true, id: refID}, key, props}, nil
}

// walkClientRef serializes a bare client-component reference used as a plain
// value (e.g. a component passed as a prop, like an error boundary's fallback).
// React encodes this as a by-value reference "$<hex>" to the import row (unlike
// the element-type position, which uses the lazy "$L<hex>").
func (r *reconciler) walkClientRef(v gojalib.Value) (any, error) {
	id, err := r.callString("clientId", v)
	if err != nil {
		return nil, err
	}
	moduleID, export := splitModuleExport(id)
	refID, err := r.clientRefID(moduleID, export)
	if err != nil {
		return nil, err
	}
	return ref{lazy: false, id: refID}, nil
}

// walkFragment serializes a Fragment as ["$", "$Sreact.fragment", key, props].
func (r *reconciler) walkFragment(v gojalib.Value) (any, error) {
	return r.walkSymbolElement(v, "react.fragment")
}

// walkSuspense serializes a Suspense boundary as ["$", "$Sreact.suspense", key,
// props]. Async children inside props become "$L<id>" lazy refs (via deferAsync)
// whose resolved rows stream later, so the client shows the fallback meanwhile.
func (r *reconciler) walkSuspense(v gojalib.Value) (any, error) {
	return r.walkSymbolElement(v, "react.suspense")
}

// walkSymbolElement serializes an element whose type is a well-known React symbol.
func (r *reconciler) walkSymbolElement(v gojalib.Value, symbol string) (any, error) {
	obj := v.ToObject(r.rt)
	key, err := r.walkValue(obj.Get("key"))
	if err != nil {
		return nil, err
	}
	props, err := r.walkProps(obj.Get("props"))
	if err != nil {
		return nil, err
	}
	return []any{elementMarker{}, symbolRef{name: symbol}, key, props}, nil
}

// deferAsync records a pending async subtree and returns a lazy ref to its future
// row. The promise's resolved value is serialized as model row id during drain.
func (r *reconciler) deferAsync(v gojalib.Value) (any, error) {
	id := r.fw.nextID()
	r.pending = append(r.pending, pendingNode{id: id, promise: v.ToObject(r.rt)})
	return ref{lazy: true, id: id}, nil
}

// walkProps walks an element's props object into a model map (children included).
func (r *reconciler) walkProps(v gojalib.Value) (any, error) {
	if v == nil || gojalib.IsUndefined(v) || gojalib.IsNull(v) {
		return map[string]any{}, nil
	}
	return r.walkObject(v)
}

// walkObject walks a plain JS object into a model map, skipping ref and
// non-serializable (function/undefined) values.
func (r *reconciler) walkObject(v gojalib.Value) (any, error) {
	obj := v.ToObject(r.rt)
	keys := obj.Keys()
	m := make(map[string]any, len(keys))
	for _, k := range keys {
		if k == "ref" {
			continue
		}
		mv, err := r.walkValue(obj.Get(k))
		if err != nil {
			return nil, err
		}
		m[k] = mv
	}
	return m, nil
}

// walkArray walks a JS array into a model slice.
func (r *reconciler) walkArray(v gojalib.Value) (any, error) {
	obj := v.ToObject(r.rt)
	n := int(obj.Get("length").ToInteger())
	arr := make([]any, 0, n)
	for i := 0; i < n; i++ {
		ev, err := r.walkValue(obj.Get(strconv.Itoa(i)))
		if err != nil {
			return nil, err
		}
		arr = append(arr, ev)
	}
	return arr, nil
}

// clientRefID returns the import-row id for a client module export, emitting the
// import row on first use and de-duplicating thereafter.
func (r *reconciler) clientRefID(moduleID, export string) (int, error) {
	key := moduleID + "#" + export
	if id, ok := r.clientRefs[key]; ok {
		return id, nil
	}
	url := r.importURLs[moduleID]
	if url == "" {
		if r.logger != nil {
			r.logger.Warn("nativersc: no chunk URL for client module", "moduleID", moduleID)
		}
		url = moduleID // best effort; the client will fail to import, but loudly
	}
	id := r.fw.nextID()
	if err := r.fw.writeImport(id, url, export); err != nil {
		return 0, err
	}
	r.clientRefs[key] = id
	return id, nil
}

// call invokes an __rsc__ method with the node as its sole argument.
func (r *reconciler) call(method string, args ...gojalib.Value) (gojalib.Value, error) {
	fn, ok := gojalib.AssertFunction(r.rsc.Get(method))
	if !ok {
		return nil, fmt.Errorf("nativersc: %s.%s is not a function", rscHelper, method)
	}
	return fn(r.rsc, args...)
}

// callString invokes an __rsc__ method and returns its result as a string.
func (r *reconciler) callString(method string, args ...gojalib.Value) (string, error) {
	v, err := r.call(method, args...)
	if err != nil {
		return "", err
	}
	return v.String(), nil
}

// escapeUserString escapes a user string that begins with "$" so the client does
// not mistake it for a reference ("$5.00" -> "$$5.00").
func escapeUserString(s string) string {
	if strings.HasPrefix(s, "$") {
		return "$" + s
	}
	return s
}

// splitModuleExport splits a client reference id "moduleID#export" on the LAST
// "#" so module paths containing "#" still resolve.
func splitModuleExport(id string) (moduleID, export string) {
	if i := strings.LastIndex(id, "#"); i >= 0 {
		return id[:i], id[i+1:]
	}
	return id, "default"
}
