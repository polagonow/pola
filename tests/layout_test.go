package tests

import "testing"

// ─── Route-group layout presence ──────────────────────────────────────────────

func TestRSC_BlogLayout_WrapsPostsPage(t *testing.T) {
	body := rsc(t, "/posts")
	// (blog)/layout.tsx contributes the sidebar; (blog)/posts/layout.tsx adds the section badge.
	if !flightContains(body, "Topics") {
		t.Error("Posts page missing (blog) sidebar layout content 'Topics'")
	}
	if !flightContains(body, "Blog") {
		t.Error("Posts page missing (blog)/posts section badge 'Blog'")
	}
}

func TestRSC_AboutLayout_WrapsPage(t *testing.T) {
	body := rsc(t, "/about")
	if !flightContains(body, "ℹ About") {
		t.Errorf("expected about layout badge 'ℹ About' in RSC stream, got:\n%s", body[:min(len(body), 400)])
	}
}

func TestRSC_ProfileLayout_WrapsPage(t *testing.T) {
	body := rsc(t, "/profile")
	if !flightContains(body, "👤 Profile") {
		t.Errorf("expected profile layout badge '👤 Profile' in RSC stream, got:\n%s", body[:min(len(body), 400)])
	}
}

func TestRSC_PostLayout_WrapsPostDetail(t *testing.T) {
	body := rsc(t, "/posts/go-react-ssr")
	if !flightContains(body, "← All posts") {
		t.Errorf("expected post layout back-link '← All posts' in RSC stream, got:\n%s", body[:min(len(body), 400)])
	}
}

func TestRSC_ProjectLayout_WrapsProjectDetail(t *testing.T) {
	body := rsc(t, "/projects/1")
	if !flightContains(body, "← All projects") {
		t.Errorf("expected project layout back-link '← All projects' in RSC stream, got:\n%s", body[:min(len(body), 400)])
	}
}

// ─── Loading-state tests ───────────────────────────────────────────────────────

// Loading components are Suspense fallbacks that appear only when async data is
// delayed. In tests, bridge functions resolve immediately, so real content is
// always streamed — confirming the page is NOT stuck in the loading state.

func TestRSC_PostsPage_NoLoadingSkeleton(t *testing.T) {
	body := rsc(t, "/posts")
	if !flightContains(body, "Building SSR with Go and React") {
		t.Error("PostsPage should show real content, not loading skeleton")
	}
}

func TestRSC_ProjectsPage_NoLoadingSkeleton(t *testing.T) {
	body := rsc(t, "/projects")
	if !flightContains(body, "GoJSX") {
		t.Error("ProjectsPage should show real content, not loading skeleton")
	}
}
