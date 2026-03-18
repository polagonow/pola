// Package filesystem provides a static file server backed by the OS filesystem.
package filesystem

import (
	"net/http"

	"github.com/polagonow/pola/core"
)

type server struct{ dir string }

// New creates a static file server rooted at dir.
func New(dir string) core.AssetServer { return &server{dir: dir} }

func (s *server) Handler(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(s.dir)))
}
