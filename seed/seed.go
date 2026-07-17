// Package seed provides a Rails-style database seeding hook.
//
// A project defines a db/seeds package exporting:
//
//	func Seed(ctx context.Context, r *core.Registry) error
//
// The pola CLI's dbseed autoload wires it into the app via a generated
// pola_seed.go overlay (which calls seed.Register from an init()), and
// `pola db seed` runs the app with POLA_SEED_ONLY=true so pola.Ready() invokes
// the registered seeders (with the full DI registry available) and exits.
package seed

import (
	"context"

	"github.com/polagonow/pola/core"
)

// Func populates the database. It receives the app's DI registry so it can
// resolve services and repositories via core.MustInvoke.
type Func func(ctx context.Context, r *core.Registry) error

var registered []Func

// Register adds a seed function. The generated pola_seed.go overlay calls this
// from an init() when a db/seeds package is present. nil functions are ignored.
func Register(fn Func) {
	if fn != nil {
		registered = append(registered, fn)
	}
}

// HasSeeds reports whether any seed functions have been registered.
func HasSeeds() bool { return len(registered) > 0 }

// Run executes all registered seed functions in registration order, stopping at
// the first error.
func Run(ctx context.Context, r *core.Registry) error {
	for _, fn := range registered {
		if err := fn(ctx, r); err != nil {
			return err
		}
	}
	return nil
}
