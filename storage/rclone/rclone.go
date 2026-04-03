// Package rclone implements a storage driver backed by the rclone Go library.
// It supports any backend rclone supports (S3, GCS, Azure, etc.).
//
// Users must import the desired rclone backends in their application:
//
//	_ "github.com/rclone/rclone/backend/local"
//	_ "github.com/rclone/rclone/backend/s3"
//	_ "github.com/rclone/rclone/backend/googlecloudstorage"
//
// Or import all backends at once:
//
//	_ "github.com/rclone/rclone/backend/all"
package rclone

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/polagonow/pola/storage"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/fs/walk"
)

// Storage is an rclone-backed storage driver.
type Storage struct {
	remote string
}

// NewStorage creates a new rclone storage driver.
//
// For inline backends, driver is the rclone backend name (e.g. "s3", "local")
// and location is the bucket or path:
//
//	rclone.NewStorage("s3", "my-bucket")
//	rclone.NewStorage("local", "/tmp/uploads")
//
// For named remotes (configured via `rclone config`), pass an empty driver
// and use the "remote:path" format for location:
//
//	rclone.NewStorage("", "myremote:bucket/path")
func NewStorage(driver, location string) *Storage {
	remote := location
	if driver != "" {
		remote = fmt.Sprintf(":%s:%s", driver, location)
	}
	return &Storage{remote: remote}
}

func (r *Storage) newFs(ctx context.Context) (fs.Fs, error) {
	dstFs, err := fs.NewFs(ctx, r.remote)
	if err != nil {
		return nil, fmt.Errorf("failed to create fs: %w", err)
	}
	return dstFs, nil
}

// Save writes content to path using rclone's Rcat.
func (r *Storage) Save(ctx context.Context, content io.Reader, path string) error {
	dstFs, err := r.newFs(ctx)
	if err != nil {
		return err
	}

	_, err = operations.Rcat(ctx, dstFs, path, io.NopCloser(content), time.Now(), nil)
	if err != nil {
		return fmt.Errorf("rcat failed: %w", err)
	}
	return nil
}

// Stat returns metadata about the file at path.
func (r *Storage) Stat(ctx context.Context, path string) (*storage.Stat, error) {
	dstFs, err := r.newFs(ctx)
	if err != nil {
		return nil, err
	}

	obj, err := dstFs.NewObject(ctx, path)
	if err != nil {
		if errors.Is(err, fs.ErrorObjectNotFound) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("stat failed: %w", err)
	}

	return &storage.Stat{
		ModifiedTime: obj.ModTime(ctx),
		Size:         obj.Size(),
		Name:         obj.Remote(),
		Path:         path,
	}, nil
}

// Open returns a reader for the file at path.
func (r *Storage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	dstFs, err := r.newFs(ctx)
	if err != nil {
		return nil, err
	}

	obj, err := dstFs.NewObject(ctx, path)
	if err != nil {
		if errors.Is(err, fs.ErrorObjectNotFound) {
			return nil, storage.ErrNotExist
		}
		return nil, fmt.Errorf("open failed: %w", err)
	}

	return obj.Open(ctx)
}

// Delete removes the file at path.
func (r *Storage) Delete(ctx context.Context, path string) error {
	dstFs, err := r.newFs(ctx)
	if err != nil {
		return err
	}

	obj, err := dstFs.NewObject(ctx, path)
	if err != nil {
		if errors.Is(err, fs.ErrorObjectNotFound) {
			return storage.ErrNotExist
		}
		return fmt.Errorf("delete failed: %w", err)
	}

	return operations.DeleteFile(ctx, obj)
}

// List returns metadata for all files under path.
func (r *Storage) List(ctx context.Context, path string) ([]*storage.Stat, error) {
	dstFs, err := r.newFs(ctx)
	if err != nil {
		return nil, err
	}

	var entries fs.DirEntries
	err = walk.ListR(ctx, dstFs, path, true, -1, walk.ListAll, func(e fs.DirEntries) error {
		entries = append(entries, e...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}

	var stats []*storage.Stat
	for _, entry := range entries {
		if obj, ok := entry.(fs.Object); ok {
			stats = append(stats, &storage.Stat{
				ModifiedTime: obj.ModTime(ctx),
				Size:         obj.Size(),
				Name:         obj.Remote(),
				Path:         obj.Remote(),
			})
		}
	}
	return stats, nil
}
