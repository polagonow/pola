package actions

import (
	"fmt"
	"time"
)

// Blog exposes blog data to JavaScript via the Pola bridge.
//
//	import { Blog } from "@pola/di"
//	const posts = await Blog.getPosts()
type Blog struct{}

var revisions = map[string][]map[string]any{
	"go-react-ssr": {
		{"rev": "v3", "date": "2024-01-15", "summary": "Published — added Suspense streaming section."},
		{"rev": "v2", "date": "2024-01-10", "summary": "Draft — expanded esbuild two-pass explanation."},
		{"rev": "v1", "date": "2024-01-05", "summary": "Initial draft — skeleton outline only."},
	},
	"rsc-deep-dive": {
		{"rev": "v2", "date": "2024-02-03", "summary": "Published — Flight wire-format diagrams added."},
		{"rev": "v1", "date": "2024-01-28", "summary": "Initial draft — protocol walkthrough."},
	},
	"goja-vm-internals": {
		{"rev": "v1", "date": "2024-03-10", "summary": "Published — first and only revision."},
	},
}

var posts = []map[string]any{
	{"id": 1, "slug": "go-react-ssr", "title": "Building SSR with Go and React",
		"excerpt": "How to run React Server Components inside a Go process using Goja.",
		"author":  "Jane Doe", "date": "2024-01-15", "readTime": 5,
		"tags": []any{"go", "react", "ssr"}},
	{"id": 2, "slug": "rsc-deep-dive", "title": "React Server Components Deep Dive",
		"excerpt": "Understanding the Flight wire protocol and how RSC trees serialize.",
		"author":  "Jane Doe", "date": "2024-02-03", "readTime": 8,
		"tags": []any{"react", "rsc", "performance"}},
	{"id": 3, "slug": "goja-vm-internals", "title": "Goja VM Internals",
		"excerpt": "A tour through the event loop, promise scheduling, and Go↔JS bridging.",
		"author":  "Jane Doe", "date": "2024-03-10", "readTime": 12,
		"tags": []any{"go", "javascript", "vm"}},
}

var projects = []map[string]any{
	{"id": "1", "title": "GoJSX", "description": "Go-powered React SSR framework.",
		"tech": []any{"Go", "React", "TypeScript", "esbuild"}, "stars": 142, "status": "active"},
	{"id": "2", "title": "GojaBridge", "description": "Type-safe Go ↔ JS bridge.",
		"tech": []any{"Go", "Goja"}, "stars": 38, "status": "stable"},
	{"id": "3", "title": "FlightDecode", "description": "Pure-Go Flight wire-format decoder.",
		"tech": []any{"Go", "React"}, "stars": 21, "status": "beta"},
}

func (b *Blog) GetPosts() ([]map[string]any, error) {
	return posts, nil
}

func (b *Blog) GetPost(slug string) (map[string]any, error) {
	fmt.Printf("Fetching post with slug %q...\n", slug)
	for _, p := range posts {
		if p["slug"] == slug {
			return p, nil
		}
	}
	return nil, fmt.Errorf("post %q not found", slug)
}

func (b *Blog) GetProjects() ([]map[string]any, error) {
	time.Sleep(1 * time.Second) // simulate latency
	return projects, nil
}

func (b *Blog) GetProject(id string) (map[string]any, error) {
	for _, p := range projects {
		if p["id"] == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", id)
}

func (b *Blog) GetRevisions(slug string) ([]map[string]any, error) {
	if revs, ok := revisions[slug]; ok {
		return revs, nil
	}
	return nil, fmt.Errorf("no revisions for post %q", slug)
}

func (b *Blog) GetRevision(slug string, rev string) (map[string]any, error) {
	for _, r := range revisions[slug] {
		if r["rev"] == rev {
			return r, nil
		}
	}
	return nil, fmt.Errorf("revision %q not found for post %q", rev, slug)
}

func (b *Blog) GetProfile() (map[string]any, error) {
	return map[string]any{
		"id": "1", "name": "Jane Doe", "email": "jane@example.com",
		"role": "Senior Engineer", "bio": "Building dev tools.",
		"github": "janedoe", "website": "https://janedoe.dev",
	}, nil
}

func (b *Blog) TriggerError(msg string) error {
	if msg == "" {
		msg = "Forced error for testing"
	}
	return fmt.Errorf("%s", msg)
}
