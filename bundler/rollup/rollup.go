// Package rollup provides a Rollup bundler stub for the Pola framework.
// It implements core.Bundler but returns ErrNotImplemented for all operations.
// This stub exists to satisfy interface compliance; a full implementation is
// planned for a future release.
package rollup

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
)

// ErrNotImplemented is returned by all Bundler methods in this stub.
var ErrNotImplemented = errors.New("rollup: not implemented")

// Bundler is a stub implementation of core.Bundler backed by Rollup.
type Bundler struct{}

// New returns a new Rollup Bundler stub.
func New() *Bundler { return &Bundler{} }

// Name returns the bundler name.
func (b *Bundler) Name() string { return "rollup" }

// Build always returns ErrNotImplemented.
func (b *Bundler) Build(_ context.Context, _ core.BundleInput) (*core.BundleOutput, error) {
	return nil, ErrNotImplemented
}

// Watch always returns ErrNotImplemented.
func (b *Bundler) Watch(_ context.Context, _ core.BundleInput, _ func(*core.BundleOutput)) error {
	return ErrNotImplemented
}
