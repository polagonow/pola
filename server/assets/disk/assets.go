// Package disk implements framework.AssetServer by serving files from disk.
package disk

import "net/http"

// DiskAssetServer implements framework.AssetServer by serving files from disk.
type DiskAssetServer struct {
	dir string
}

// NewDiskAssetServer creates an AssetServer backed by the given directory.
func NewDiskAssetServer(dir string) *DiskAssetServer {
	return &DiskAssetServer{dir: dir}
}

// Handler returns an http.Handler that strips prefix and serves from disk.
func (a *DiskAssetServer) Handler(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(a.dir)))
}
