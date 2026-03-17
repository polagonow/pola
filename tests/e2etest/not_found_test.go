package e2etest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// assertNotFoundError verifies that a Flight body contains an error row and
// that the error digest contains "not found" — confirming the page threw a
// data-missing error rather than an unrelated server error.
func assertNotFoundError(t *testing.T, body string) {
	t.Helper()
	if !strings.Contains(body, ":E{") && !strings.Contains(body, ":E\"") {
		t.Errorf("expected Flight error row (:E), got:\n%s", body[:min(len(body), 300)])
	}
	if !strings.Contains(strings.ToLower(body), "not found") {
		t.Errorf("expected \"not found\" in error digest, got:\n%s", body[:min(len(body), 300)])
	}
}

// ─── Global not-found (no route matches) ──────────────────────────────────────

func TestGlobalNotFound_RSC_Returns404(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		status, _ := RSCAny(t, f, "/nonexistent-route")
		if status != http.StatusNotFound {
			t.Errorf("expected HTTP 404 for unknown RSC route, got %d", status)
		}
	})
}

func TestGlobalNotFound_RSC_ContentType(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent-route", nil)
		req.Header.Set("Content-Type", "text/x-component")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		ct := w.Result().Header.Get("Content-Type")
		if !strings.Contains(ct, "text/x-component") {
			t.Errorf("expected text/x-component for RSC 404, got %q", ct)
		}
	})
}

func TestGlobalNotFound_RSC_HasFlightRow(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		_, body := RSCAny(t, f, "/nonexistent-route")
		if !strings.Contains(body, "0:") {
			t.Errorf("expected Flight 0: row in global not-found RSC response, got:\n%s", body[:min(len(body), 300)])
		}
	})
}

func TestGlobalNotFound_RSC_Content(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		_, body := RSCAny(t, f, "/nonexistent-route")
		for _, want := range []string{"Page not found", "Go home"} {
			if !strings.Contains(body, want) {
				t.Errorf("global not-found RSC missing %q, got:\n%s", want, body[:min(len(body), 400)])
			}
		}
	})
}

func TestGlobalNotFound_HTML_Returns404(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent-route", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Result().StatusCode != http.StatusNotFound {
			t.Errorf("expected HTTP 404 for unknown HTML route, got %d", w.Result().StatusCode)
		}
	})
}

func TestGlobalNotFound_HTML_IsClientShell(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		app := f.getApp(t)
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent-route", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		body, _ := io.ReadAll(w.Result().Body)
		// The HTML 404 response is the standard client shell (not a raw error page).
		// The browser bootstraps and makes a subsequent RSC request to fetch the
		// actual GlobalNotFound component content.
		if !strings.Contains(string(body), `id="root"`) {
			t.Errorf("global not-found HTML response should be client shell with id=\"root\", got:\n%s",
				string(body)[:min(len(string(body)), 400)])
		}
	})
}

// ─── Per-page not-found (route matches, data missing → error boundary fires) ──

func TestPageNotFound_Post(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		assertNotFoundError(t, RSC(t, f, "/posts/no-such-post"))
	})
}

func TestPageNotFound_Project(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		assertNotFoundError(t, RSC(t, f, "/projects/no-such-id"))
	})
}

func TestPageNotFound_Revision(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		assertNotFoundError(t, RSC(t, f, "/posts/go-react-ssr/revisions/v99"))
	})
}
