package suite

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gojsx/test/fixture"
)

// RunHTMLShellTests verifies the HTML document shell returned for any route.
func RunHTMLShellTests(t *testing.T) {
	t.Helper()

	t.Run("HasMountPoint", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.Page(t, f, "/")
			if !strings.Contains(body, `id="root"`) {
				t.Error("HTML shell missing mount point <div id=\"root\">")
			}
		})
	})

	t.Run("ServesClientEntryScript", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			body := fixture.Page(t, f, "/")
			if !strings.Contains(body, `type="module"`) {
				t.Error("HTML shell missing <script type=\"module\"> client entry point")
			}
		})
	})

	t.Run("ReturnsHTMLContentType", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			ct := w.Result().Header.Get("Content-Type")
			if !strings.Contains(ct, "text/html") {
				t.Errorf("expected text/html Content-Type, got %q", ct)
			}
		})
	})

	t.Run("UnknownRouteReturns404", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent", nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
			if w.Result().StatusCode != http.StatusNotFound {
				t.Errorf("expected 404 for unregistered route, got %d", w.Result().StatusCode)
			}
		})
	})
}
