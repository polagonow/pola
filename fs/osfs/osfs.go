// Package osfs provides an FS implementation backed by the OS file system.
// It supports hot reload via fsnotify.
package osfs

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	samberdo "github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

// OSFS is an FS rooted at a directory on the OS filesystem.
type OSFS struct{ root string }

// New creates an OSFS rooted at dir.
func New(root string) *OSFS { return &OSFS{root: root} }

func (fs *OSFS) Name() string { return "osfs" }

// resolve returns the OS path for a given FS path. Absolute paths are returned
// as-is; relative paths are joined with the FS root.
func (fs *OSFS) resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(fs.root, path)
}

func (fs *OSFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(fs.resolve(path))
}

func (fs *OSFS) ReadDir(path string) ([]core.FSFileInfo, error) {
	entries, err := os.ReadDir(fs.resolve(path))
	if err != nil {
		return nil, err
	}
	out := make([]core.FSFileInfo, len(entries))
	for i, e := range entries {
		info, _ := e.Info()
		var sz int64
		var mt time.Time
		if info != nil {
			sz = info.Size()
			mt = info.ModTime()
		}
		out[i] = core.FSFileInfo{Name: e.Name(), IsDir: e.IsDir(), Size: sz, ModTime: mt}
	}
	return out, nil
}

func (fs *OSFS) Exists(path string) bool {
	_, err := os.Stat(fs.resolve(path))
	return err == nil
}

func (fs *OSFS) Watch(path string, onChange func(string)) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	watchPath := path
	if !filepath.IsAbs(path) {
		watchPath = filepath.Join(fs.root, path)
	}
	// Add the root and every non-excluded subdirectory recursively.
	if err := filepath.WalkDir(watchPath, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			name := filepath.Base(p)
			if name == "node_modules" || name == "public" || (name != "." && strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return w.Add(p)
		}
		return nil
	}); err != nil {
		w.Close()
		return err
	}
	go func() {
		defer w.Close()
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
					onChange(event.Name)
				}
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return nil
}

func init() {
	di.Stage(func(i samberdo.Injector) {
		samberdo.Provide(i, func(_ samberdo.Injector) (core.FS, error) {
			return New("."), nil
		})
		samberdo.Provide(i, func(_ samberdo.Injector) (core.AssetServerFactory, error) {
			return func(publicDir string) core.AssetServer {
				return &osAssetServer{dir: publicDir}
			}, nil
		})
	})
}

type osAssetServer struct{ dir string }

func (s *osAssetServer) Handler(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(s.dir)))
}
