// Package osfs provides an FS implementation backed by the OS file system.
// It supports hot reload via fsnotify.
package osfs

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/polagonow/pola/core"
)

// OSFS is an FS rooted at a directory on the OS filesystem.
type OSFS struct{ root string }

// New creates an OSFS rooted at dir.
func New(root string) *OSFS { return &OSFS{root: root} }

func (fs *OSFS) Name() string { return "osfs" }

func (fs *OSFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(fs.root, path))
}

func (fs *OSFS) ReadDir(path string) ([]core.FSFileInfo, error) {
	entries, err := os.ReadDir(filepath.Join(fs.root, path))
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
	_, err := os.Stat(filepath.Join(fs.root, path))
	return err == nil
}

func (fs *OSFS) Watch(path string, onChange func(string)) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(filepath.Join(fs.root, path)); err != nil {
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
				onChange(event.Name)
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
	core.RegisterFS(func() core.FS { return New(".") })
	core.RegisterAssetServer(func(publicDir string) core.AssetServer {
		return &osAssetServer{dir: publicDir}
	})
}

type osAssetServer struct{ dir string }

func (s *osAssetServer) Handler(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(s.dir)))
}
