// Package disk implements framework.AssetServer by serving files from disk.
package disk

import "net/http"

// AssetServer implements framework.AssetServer by serving files from disk.
type AssetServer struct {
	dir string
}

// NewAssetServer creates an AssetServer backed by the given directory.
func NewAssetServer(dir string) *AssetServer {
	return &AssetServer{dir: dir}
}

// Handler returns an http.Handler that strips prefix and serves from disk.
func (a *AssetServer) Handler(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(a.dir)))
}
