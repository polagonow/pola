package mcp

import (
	"net/http"
	"strings"
)

// mcpMiddleware short-circuits requests under the configured mount path to
// the MCP HTTP handler. All other requests pass through to next.
//
// It is registered ahead of the framework's request logger, recovery, and
// CSRF middlewares so MCP requests bypass them — SSE streams would
// otherwise flood logs, and CSRF would 403 every MCP POST.
type mcpMiddleware struct {
	mount   string
	handler http.Handler
}

func (m *mcpMiddleware) Name() string { return "mcp" }

func (m *mcpMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if matchesMount(r.URL.Path, m.mount) {
			m.handler.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// matchesMount returns true if path is the mount or sits under it
// (i.e. it equals mount or has mount as a path-segment prefix).
// "/mcp" matches "/mcp" and "/mcp/foo" but not "/mcpish".
func matchesMount(path, mount string) bool {
	if path == mount {
		return true
	}
	return strings.HasPrefix(path, mount+"/")
}
