// Package fs implements a local filesystem storage driver.
package fs

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/polagonow/pola/storage"
)

// Storage is a local filesystem storage driver.
type Storage struct {
	root string
}

// Config holds configuration for the filesystem driver.
type Config struct {
	// Root is the base directory for stored files.
	Root string
}

// NewStorage returns a new filesystem storage rooted at cfg.Root.
func NewStorage(cfg Config) *Storage {
	return &Storage{root: cfg.Root}
}

func (s *Storage) abs(path string) (string, error) {
	abs := filepath.Join(s.root, path)
	root := filepath.Clean(s.root)
	if abs != root && !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes storage root", path)
	}
	return abs, nil
}

// Save writes content to path, creating parent directories as needed.
func (s *Storage) Save(_ context.Context, content io.Reader, path string) error {
	abs, err := s.abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}

	w, err := os.Create(abs)
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, content); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

// Stat returns file metadata. Returns storage.ErrNotExist if not found.
func (s *Storage) Stat(_ context.Context, path string) (*storage.Stat, error) {
	abs, err := s.abs(path)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(abs)
	if os.IsNotExist(err) {
		return nil, storage.ErrNotExist
	} else if err != nil {
		return nil, err
	}

	return &storage.Stat{
		ModifiedTime: fi.ModTime(),
		Size:         fi.Size(),
		ContentType:  mime.TypeByExtension(filepath.Ext(fi.Name())),
		Name:         fi.Name(),
		Path:         path,
	}, nil
}

// Open opens the file at path for reading. Returns storage.ErrNotExist if not found.
func (s *Storage) Open(_ context.Context, path string) (io.ReadCloser, error) {
	abs, err := s.abs(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(abs)
	if os.IsNotExist(err) {
		return nil, storage.ErrNotExist
	}
	return f, err
}

// Delete removes the file at path.
func (s *Storage) Delete(_ context.Context, path string) error {
	abs, err := s.abs(path)
	if err != nil {
		return err
	}
	return os.Remove(abs)
}

// List returns metadata for all files in path.
func (s *Storage) List(_ context.Context, path string) ([]*storage.Stat, error) {
	abs, err := s.abs(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(abs)
	if os.IsNotExist(err) {
		return nil, storage.ErrNotExist
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	fis, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	stats := make([]*storage.Stat, 0, len(fis))
	for _, fi := range fis {
		stats = append(stats, &storage.Stat{
			Name:         fi.Name(),
			Path:         filepath.Join(path, fi.Name()),
			Size:         fi.Size(),
			ModifiedTime: fi.ModTime(),
			ContentType:  mime.TypeByExtension(filepath.Ext(fi.Name())),
		})
	}
	return stats, nil
}
