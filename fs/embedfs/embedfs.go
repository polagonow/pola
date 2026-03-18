// Package embedfs provides an FS implementation backed by embed.FS.
// Hot reload (Watch) is a no-op since embedded files are immutable.
package embedfs

import (
	"embed"
	"time"

	"github.com/polagonow/pola/core"
)

// EmbedFS wraps an embed.FS to satisfy core.FS.
type EmbedFS struct{ inner embed.FS }

// New creates an EmbedFS wrapping the given embed.FS.
func New(f embed.FS) *EmbedFS { return &EmbedFS{inner: f} }

func (e *EmbedFS) Name() string { return "embedfs" }

func (e *EmbedFS) ReadFile(path string) ([]byte, error) { return e.inner.ReadFile(path) }

func (e *EmbedFS) ReadDir(path string) ([]core.FSFileInfo, error) {
	entries, err := e.inner.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]core.FSFileInfo, len(entries))
	for i, entry := range entries {
		var sz int64
		var mt time.Time
		if info, err := entry.Info(); err == nil {
			sz = info.Size()
			mt = info.ModTime()
		}
		out[i] = core.FSFileInfo{Name: entry.Name(), IsDir: entry.IsDir(), Size: sz, ModTime: mt}
	}
	return out, nil
}

func (e *EmbedFS) Exists(path string) bool {
	_, err := e.inner.Open(path)
	return err == nil
}

// Watch is a no-op for embedded assets (immutable at runtime).
func (e *EmbedFS) Watch(path string, onChange func(string)) error { return nil }
