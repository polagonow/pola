// Package layered provides a virtual filesystem overlay that combines an
// inner core.FS with in-memory generated files. Generated files take
// precedence over the inner FS on reads.
package layered

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/polagonow/pola/core"
)

// FS wraps an inner core.FS and overlays in-memory virtual files on top.
// Virtual files take precedence: if a path exists in both the overlay and the
// inner FS, the overlay wins. Thread-safe for concurrent reads and writes.
type FS struct {
	inner core.FS

	mu    sync.RWMutex
	files map[string][]byte // abs path → content
}

// New creates a layered FS wrapping inner.
func New(inner core.FS) *FS {
	return &FS{
		inner: inner,
		files: make(map[string][]byte),
	}
}

func (fs *FS) Name() string { return "layered(" + fs.inner.Name() + ")" }

// Set registers a virtual file at the given absolute path. Subsequent
// ReadFile and Exists calls for this path will return the virtual content.
func (fs *FS) Set(path string, content []byte) {
	fs.mu.Lock()
	fs.files[path] = content
	fs.mu.Unlock()
}

// Remove deletes a virtual file. Subsequent reads fall through to the inner FS.
func (fs *FS) Remove(path string) {
	fs.mu.Lock()
	delete(fs.files, path)
	fs.mu.Unlock()
}

func (fs *FS) ReadFile(path string) ([]byte, error) {
	fs.mu.RLock()
	data, ok := fs.files[path]
	fs.mu.RUnlock()
	if ok {
		return data, nil
	}
	return fs.inner.ReadFile(path)
}

func (fs *FS) ReadDir(path string) ([]core.FSFileInfo, error) {
	entries, err := fs.inner.ReadDir(path)
	if err != nil {
		entries = nil
	}

	// Merge virtual files that live directly under this directory.
	fs.mu.RLock()
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		seen[e.Name] = true
	}
	absPath, _ := filepath.Abs(path)
	for vpath, content := range fs.files {
		dir := filepath.Dir(vpath)
		if dir != absPath && dir != path {
			continue
		}
		name := filepath.Base(vpath)
		if seen[name] {
			// Virtual file overrides — replace the entry with virtual metadata.
			for i, e := range entries {
				if e.Name == name {
					entries[i] = core.FSFileInfo{
						Name:    name,
						IsDir:   false,
						Size:    int64(len(content)),
						ModTime: time.Now(),
					}
					break
				}
			}
		} else {
			entries = append(entries, core.FSFileInfo{
				Name:    name,
				IsDir:   false,
				Size:    int64(len(content)),
				ModTime: time.Now(),
			})
			seen[name] = true
		}
	}
	fs.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	if len(entries) == 0 && err != nil {
		return nil, err
	}
	return entries, nil
}

func (fs *FS) Exists(path string) bool {
	fs.mu.RLock()
	_, ok := fs.files[path]
	fs.mu.RUnlock()
	if ok {
		return true
	}
	return fs.inner.Exists(path)
}

func (fs *FS) Watch(path string, onChange func(string)) error {
	return fs.inner.Watch(path, onChange)
}

// VirtualPaths returns a sorted list of all registered virtual file paths.
func (fs *FS) VirtualPaths() []string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	paths := make([]string, 0, len(fs.files))
	for p := range fs.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// String returns a human-readable summary of the layered FS.
func (fs *FS) String() string {
	paths := fs.VirtualPaths()
	if len(paths) == 0 {
		return fmt.Sprintf("layered(%s, 0 virtual files)", fs.inner.Name())
	}
	var b strings.Builder
	fmt.Fprintf(&b, "layered(%s, %d virtual files):", fs.inner.Name(), len(paths))
	for _, p := range paths {
		b.WriteString("\n  ")
		b.WriteString(p)
	}
	return b.String()
}
