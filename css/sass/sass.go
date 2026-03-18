// Package sass provides a CSS plugin stub for SASS/SCSS processing.
// It is not yet implemented and returns ErrNotImplemented for all operations.
package sass

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
)

// ErrNotImplemented is returned by all Sass operations.
var ErrNotImplemented = errors.New("sass: not implemented")

// Sass is a stub CSS processor for SASS/SCSS.
type Sass struct{}

// New returns a Sass CSS processor stub.
func New() core.CSS { return &Sass{} }

// Ensure Sass satisfies core.CSS.
var _ core.CSS = (*Sass)(nil)

// Name returns the CSS implementation name.
func (s *Sass) Name() string { return "sass" }

// Process always returns ErrNotImplemented.
func (s *Sass) Process(_ context.Context, _, _ string) error { return ErrNotImplemented }

// Watch always returns ErrNotImplemented.
func (s *Sass) Watch(_ context.Context, _, _ string, _ func()) error { return ErrNotImplemented }
