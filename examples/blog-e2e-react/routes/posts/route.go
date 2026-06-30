package posts

import (
	"fmt"
	"net/http"
	"time"

	"github.com/polagonow/pola/core"
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

func (r *Route) GET(c core.Context) error {
	return c.JSON(http.StatusOK, posts)
}

func (r *Route) POST(c core.Context) error {
	var input struct {
		Title   string   `json:"title"`
		Excerpt string   `json:"excerpt"`
		Tags    []string `json:"tags"`
	}
	if err := c.ShouldBind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": "invalid JSON"})
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

	return c.JSON(http.StatusCreated, post)
}
