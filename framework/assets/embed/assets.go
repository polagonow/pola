// Package embed implements framework.AssetServer using an embedded fs.FS.
// Unlike the disk package, there is no register.go — the user must supply
// the FS and register it explicitly since it requires an application-specific
// embed.FS value.
package embed

import (
	"io/fs"
	"net/http"
)

// AssetServer implements framework.AssetServer backed by an embedded FS.
type AssetServer struct {
	fsys fs.FS
}

// NewAssetServer creates an AssetServer backed by the given fs.FS.
func NewAssetServer(fsys fs.FS) *AssetServer {
	return &AssetServer{fsys: fsys}
}

// Handler returns an http.Handler that strips prefix and serves from the
// embedded FS.
func (a *AssetServer) Handler(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.FS(a.fsys)))
}
