package tests

import (
	"strings"
	"testing"
)

// assertErrorBoundary checks that the Flight body contains an error row and that
// the real error message was passed through as the digest via onError.
func assertErrorBoundary(t *testing.T, body string) {
	t.Helper()
	hasErrorRow := strings.Contains(body, ":E{") || strings.Contains(body, ":E\"")
	if !hasErrorRow {
		t.Errorf("expected Flight error row (:E), got:\n%s", body[:min(len(body), 300)])
	}
	if !strings.Contains(body, "test-error") {
		t.Errorf("expected digest 'test-error' in Flight body, got:\n%s", body[:min(len(body), 300)])
	}
}

// assertCorrectErrorBoundary extends assertErrorBoundary by also verifying that
// the expected error-component module ID is referenced in the Flight body. This
// proves the *correct* innermost error boundary is mounted for the page — not
// just that some boundary fired.
func assertCorrectErrorBoundary(t *testing.T, body, moduleID string) {
	t.Helper()
	assertErrorBoundary(t, body)
	if !strings.Contains(body, moduleID) {
		t.Errorf("expected error module %q referenced in Flight body, got:\n%s", moduleID, body[:min(len(body), 400)])
	}
}

func TestErrorBoundary_Home(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/?error=test-error"), `"app/error"`)
}

func TestErrorBoundary_Posts(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/posts?error=test-error"), `"app/(blog)/posts/error"`)
}

func TestErrorBoundary_Projects(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/projects?error=test-error"), `"app/(work)/projects/error"`)
}

func TestErrorBoundary_ProjectDetail(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/projects/1?error=test-error"), `"app/(work)/projects/[id]/error"`)
}

func TestErrorBoundary_About(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/about?error=test-error"), `"app/about/error"`)
}

func TestErrorBoundary_Profile(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/profile?error=test-error"), `"app/profile/error"`)
}

func TestErrorBoundary_PostDetail(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/posts/go-react-ssr?error=test-error"), `"app/(blog)/posts/[slug]/error"`)
}

func TestErrorBoundary_Revisions(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/posts/go-react-ssr/revisions?error=test-error"), `"app/(blog)/posts/[slug]/revisions/error"`)
}

func TestErrorBoundary_RevisionDetail(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/posts/go-react-ssr/revisions/v1?error=test-error"), `"app/(blog)/posts/[slug]/revisions/[rev]/error"`)
}

func TestErrorBoundary_Docs(t *testing.T) {
	assertCorrectErrorBoundary(t, rsc(t, "/docs?error=test-error"), `"app/docs/[[...slug]]/error"`)
}

// ─── ImportURLs uniqueness (regression for chunk-collision fix) ────────────────

// ─── Global error boundary ────────────────────────────────────────────────────

// TestGlobalError_InImportURLs verifies that global-error.tsx is present in
// the client manifest. It is a "use client" component and must be importable
// by the browser as the outermost error boundary fallback.
func TestGlobalError_InImportURLs(t *testing.T) {
	app := requireApp(t)
	if _, ok := app.ImportURLs["app/global-error"]; !ok {
		t.Error("global-error module missing from ImportURLs")
	}
}

// TestGlobalError_ReferencedInRSCStream verifies that every page's RSC stream
// references the global-error module. Since GlobalError wraps all pages as the
// outermost __FrameworkErrorBoundary__ fallback, it is always serialised into
// the Flight stream as a module (I:) row.
func TestGlobalError_ReferencedInRSCStream(t *testing.T) {
	for _, path := range []string{"/", "/posts", "/projects", "/about", "/profile"} {
		body := rsc(t, path)
		if !strings.Contains(body, `"app/global-error"`) {
			t.Errorf("page %s: global-error module not referenced in RSC stream", path)
		}
	}
}

// ─── ImportURLs uniqueness (regression for chunk-collision fix) ────────────────

// TestImportURLs_ErrorModulesAreUnique verifies that every error module ID maps
// to a *distinct* chunk URL. This guards against the bug where multiple error.tsx
// files with the same base name all matched the same "error-*.js" chunk.
func TestImportURLs_ErrorModulesAreUnique(t *testing.T) {
	app := requireApp(t)
	errorModules := []string{
		"app/error",
		"app/(blog)/posts/error",
		"app/(work)/projects/error",
		"app/(work)/projects/[id]/error",
		"app/about/error",
		"app/profile/error",
		"app/(blog)/posts/[slug]/error",
		"app/(blog)/posts/[slug]/revisions/error",
		"app/(blog)/posts/[slug]/revisions/[rev]/error",
		"app/docs/[[...slug]]/error",
	}
	seen := make(map[string]string) // chunkURL → moduleID
	for _, id := range errorModules {
		url, ok := app.ImportURLs[id]
		if !ok {
			t.Errorf("module %q missing from importURLs", id)
			continue
		}
		if prev, dup := seen[url]; dup {
			t.Errorf("chunk collision: %q and %q both map to %q", prev, id, url)
		}
		seen[url] = id
	}
}
