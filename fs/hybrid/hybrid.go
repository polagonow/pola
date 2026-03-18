// Package hybrid provides an FS that checks embedFS first, falls back to OSFS.
// Useful for prod binaries that can still override files from disk.
package hybrid

import (
	"github.com/polagonow/pola/core"
)

// HybridFS checks embed first and falls back to os for every operation.
type HybridFS struct {
	embed core.FS
	os    core.FS
}

// New creates a HybridFS: embed checked first, osFS used as fallback.
func New(embed, os core.FS) *HybridFS { return &HybridFS{embed: embed, os: os} }

func (h *HybridFS) Name() string { return "hybrid" }

func (h *HybridFS) ReadFile(path string) ([]byte, error) {
	if data, err := h.embed.ReadFile(path); err == nil {
		return data, nil
	}
	return h.os.ReadFile(path)
}

func (h *HybridFS) ReadDir(path string) ([]core.FSFileInfo, error) {
	if entries, err := h.embed.ReadDir(path); err == nil {
		return entries, nil
	}
	return h.os.ReadDir(path)
}

func (h *HybridFS) Exists(path string) bool {
	return h.embed.Exists(path) || h.os.Exists(path)
}

func (h *HybridFS) Watch(path string, onChange func(string)) error {
	return h.os.Watch(path, onChange)
}
