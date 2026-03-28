package actions

import (
	"fmt"
	"time"
)

// Blog exposes blog data to JavaScript via the Pola bridge.
//
//	import { Blog } from "@pola/actions"
//	const posts = await Blog.getPosts()
type Blog struct{}

type Post struct {
	ID       int      `json:"id"`
	Slug     string   `json:"slug"`
	Title    string   `json:"title"`
	Excerpt  string   `json:"excerpt"`
	Author   string   `json:"author"`
	Date     string   `json:"date"`
	ReadTime int      `json:"readTime"`
	Tags     []string `json:"tags"`
}

type Project struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tech        []string `json:"tech"`
	Stars       int      `json:"stars"`
	Status      string   `json:"status"`
}

type Revision struct {
	Rev     string `json:"rev"`
	Date    string `json:"date"`
	Summary string `json:"summary"`
}

type Profile struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	Bio     string `json:"bio"`
	GitHub  string `json:"github"`
	Website string `json:"website"`
}

var revisions = map[string][]Revision{
	"go-react-ssr": {
		{Rev: "v3", Date: "2024-01-15", Summary: "Published — added Suspense streaming section."},
		{Rev: "v2", Date: "2024-01-10", Summary: "Draft — expanded esbuild two-pass explanation."},
		{Rev: "v1", Date: "2024-01-05", Summary: "Initial draft — skeleton outline only."},
	},
	"rsc-deep-dive": {
		{Rev: "v2", Date: "2024-02-03", Summary: "Published — Flight wire-format diagrams added."},
		{Rev: "v1", Date: "2024-01-28", Summary: "Initial draft — protocol walkthrough."},
	},
	"goja-vm-internals": {
		{Rev: "v1", Date: "2024-03-10", Summary: "Published — first and only revision."},
	},
}

var posts = []Post{
	{ID: 1, Slug: "go-react-ssr", Title: "Building SSR with Go and React",
		Excerpt: "How to run React Server Components inside a Go process using Goja.",
		Author: "Jane Doe", Date: "2024-01-15", ReadTime: 5,
		Tags: []string{"go", "react", "ssr"}},
	{ID: 2, Slug: "rsc-deep-dive", Title: "React Server Components Deep Dive",
		Excerpt: "Understanding the Flight wire protocol and how RSC trees serialize.",
		Author: "Jane Doe", Date: "2024-02-03", ReadTime: 8,
		Tags: []string{"react", "rsc", "performance"}},
	{ID: 3, Slug: "goja-vm-internals", Title: "Goja VM Internals",
		Excerpt: "A tour through the event loop, promise scheduling, and Go↔JS bridging.",
		Author: "Jane Doe", Date: "2024-03-10", ReadTime: 12,
		Tags: []string{"go", "javascript", "vm"}},
}

var projects = []Project{
	{ID: "1", Title: "GoJSX", Description: "Go-powered React SSR framework.",
		Tech: []string{"Go", "React", "TypeScript", "esbuild"}, Stars: 142, Status: "active"},
	{ID: "2", Title: "GojaBridge", Description: "Type-safe Go ↔ JS bridge.",
		Tech: []string{"Go", "Goja"}, Stars: 38, Status: "stable"},
	{ID: "3", Title: "FlightDecode", Description: "Pure-Go Flight wire-format decoder.",
		Tech: []string{"Go", "React"}, Stars: 21, Status: "beta"},
}

func (b *Blog) GetPosts() ([]Post, error) {
	return posts, nil
}

func (b *Blog) GetPost(slug string) (*Post, error) {
	fmt.Printf("Fetching post with slug %q...\n", slug)
	for i := range posts {
		if posts[i].Slug == slug {
			return &posts[i], nil
		}
	}
	return nil, fmt.Errorf("post %q not found", slug)
}

func (b *Blog) GetProjects() ([]Project, error) {
	time.Sleep(1 * time.Second) // simulate latency
	return projects, nil
}

func (b *Blog) GetProject(id string) (*Project, error) {
	for i := range projects {
		if projects[i].ID == id {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %q not found", id)
}

func (b *Blog) GetRevisions(slug string) ([]Revision, error) {
	if revs, ok := revisions[slug]; ok {
		return revs, nil
	}
	return nil, fmt.Errorf("no revisions for post %q", slug)
}

func (b *Blog) GetRevision(slug string, rev string) (*Revision, error) {
	for i := range revisions[slug] {
		if revisions[slug][i].Rev == rev {
			return &revisions[slug][i], nil
		}
	}
	return nil, fmt.Errorf("revision %q not found for post %q", rev, slug)
}

func (b *Blog) GetProfile() (*Profile, error) {
	return &Profile{
		ID: "1", Name: "Jane Doe", Email: "jane@example.com",
		Role: "Senior Engineer", Bio: "Building dev tools.",
		GitHub: "janedoe", Website: "https://janedoe.dev",
	}, nil
}

func (b *Blog) TriggerError(msg string) error {
	if msg == "" {
		msg = "Forced error for testing"
	}
	return fmt.Errorf("%s", msg)
}
