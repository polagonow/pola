// Package plugins registers all built-in model generators.
// Import this package with a blank import to trigger registration.
package plugins

import (
	"github.com/polagonow/pola/internal/modelgen"
	entplugin "github.com/polagonow/pola/internal/modelgen/plugins/ent"
	gormplugin "github.com/polagonow/pola/internal/modelgen/plugins/gorm"
)

func init() {
	modelgen.RegisterGenerator(&entplugin.EntGenerator{})
	modelgen.RegisterGenerator(&gormplugin.GormGenerator{})
}
