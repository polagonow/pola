package e2etest

import "testing"

// ─── Root layout ──────────────────────────────────────────────────────────────

// The root layout is a server component that imports Nav and ThemeToggle as
// client components. Their module references appear in every page's RSC stream.

func TestRSC_RootLayout_HasNav(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/")
		if !FlightContains(body, `"components/Nav"`) {
			t.Error("RSC stream missing Nav module reference in root layout")
		}
	})
}

func TestRSC_RootLayout_HasThemeToggle(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/")
		if !FlightContains(body, `"components/ThemeToggle"`) {
			t.Error("RSC stream missing ThemeToggle module reference in root layout")
		}
	})
}

// ─── Route-group layout presence ──────────────────────────────────────────────

func TestRSC_BlogLayout_WrapsPostsPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/posts")
		// (blog)/layout.tsx contributes the sidebar; (blog)/posts/layout.tsx adds the section badge.
		if !FlightContains(body, "Topics") {
			t.Error("Posts page missing (blog) sidebar layout content 'Topics'")
		}
		if !FlightContains(body, "Blog") {
			t.Error("Posts page missing (blog)/posts section badge 'Blog'")
		}
	})
}

func TestRSC_AboutLayout_WrapsPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/about")
		if !FlightContains(body, "ℹ About") {
			t.Errorf("expected about layout badge 'ℹ About' in RSC stream, got:\n%s", body[:min(len(body), 400)])
		}
	})
}

func TestRSC_ProfileLayout_WrapsPage(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/profile")
		if !FlightContains(body, "👤 Profile") {
			t.Errorf("expected profile layout badge '👤 Profile' in RSC stream, got:\n%s", body[:min(len(body), 400)])
		}
	})
}

func TestRSC_PostLayout_WrapsPostDetail(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/posts/go-react-ssr")
		if !FlightContains(body, "← All posts") {
			t.Errorf("expected post layout back-link '← All posts' in RSC stream, got:\n%s", body[:min(len(body), 400)])
		}
	})
}

func TestRSC_ProjectLayout_WrapsProjectDetail(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/projects/1")
		if !FlightContains(body, "← All projects") {
			t.Errorf("expected project layout back-link '← All projects' in RSC stream, got:\n%s", body[:min(len(body), 400)])
		}
	})
}

// ─── Loading-state tests ───────────────────────────────────────────────────────

// Loading components are Suspense fallbacks that appear only when async data is
// delayed. In tests, bridge functions resolve immediately, so real content is
// always streamed — confirming the page is NOT stuck in the loading state.

func TestRSC_PostsPage_NoLoadingSkeleton(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/posts")
		if !FlightContains(body, "Building SSR with Go and React") {
			t.Error("PostsPage should show real content, not loading skeleton")
		}
	})
}

func TestRSC_ProjectsPage_NoLoadingSkeleton(t *testing.T) {
	forEachReactApp(t, func(t *testing.T, f AppFixture) {
		body := RSC(t, f, "/projects")
		if !FlightContains(body, "GoJSX") {
			t.Error("ProjectsPage should show real content, not loading skeleton")
		}
	})
}
