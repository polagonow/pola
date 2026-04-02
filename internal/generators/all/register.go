// Package all registers all built-in generators.
// Import this package with a blank import to trigger registration.
package all

import (
	_ "github.com/polagonow/pola/internal/generators/action"
	_ "github.com/polagonow/pola/internal/generators/model"
	_ "github.com/polagonow/pola/internal/generators/route"
	_ "github.com/polagonow/pola/internal/generators/scaffold"
)
