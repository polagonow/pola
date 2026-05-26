package posts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Route handles API requests for blog posts.
//
//	GET  /posts → list all posts (JSON)
//	POST /posts → create a new post (JSON)
type Route struct{}

type Post struct {
	ID      int      `json:"id"`
	Slug    string   `json:"slug"`
	Title   string   `json:"title"`
	Excerpt string   `json:"excerpt"`
	Author  string   `json:"author"`
	Date    string   `json:"date"`
	Tags    []string `json:"tags"`
}

var posts = []Post{
	{ID: 1, Slug: "go-react-ssr", Title: "Building SSR with Go and React",
		Excerpt: "How to run React Server Components inside a Go process.",
		Author:  "Jane Doe", Date: "2024-01-15", Tags: []string{"go", "react"}},
	{ID: 2, Slug: "rsc-deep-dive", Title: "React Server Components Deep Dive",
		Excerpt: "Understanding the Flight wire protocol.",
		Author:  "Jane Doe", Date: "2024-02-03", Tags: []string{"react", "rsc"}},
}

func (r *Route) GET(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func (r *Route) POST(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Title   string   `json:"title"`
		Excerpt string   `json:"excerpt"`
		Tags    []string `json:"tags"`
	}
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	post := Post{
		ID:      len(posts) + 1,
		Slug:    fmt.Sprintf("post-%d", len(posts)+1),
		Title:   input.Title,
		Excerpt: input.Excerpt,
		Author:  "API User",
		Date:    time.Now().Format("2006-01-02"),
		Tags:    input.Tags,
	}
	posts = append(posts, post)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}
