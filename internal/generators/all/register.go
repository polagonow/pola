// Package all wires all built-in generators into the registry explicitly.
// Call Register once at CLI startup (cli.Execute).
package all

import (
	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/action"
	"github.com/polagonow/pola/internal/generators/docs"
	"github.com/polagonow/pola/internal/generators/dto"
	"github.com/polagonow/pola/internal/generators/jsbridge"
	"github.com/polagonow/pola/internal/generators/mailer"
	"github.com/polagonow/pola/internal/generators/mcp"
	"github.com/polagonow/pola/internal/generators/migration"
	"github.com/polagonow/pola/internal/generators/model"
	"github.com/polagonow/pola/internal/generators/page"
	"github.com/polagonow/pola/internal/generators/repository"
	"github.com/polagonow/pola/internal/generators/route"
	"github.com/polagonow/pola/internal/generators/scaffold"
	"github.com/polagonow/pola/internal/generators/seed"
	"github.com/polagonow/pola/internal/generators/service"
	"github.com/polagonow/pola/internal/generators/storage"
	"github.com/polagonow/pola/internal/generators/zod"
)

// Register adds every built-in generator to the registry.
func Register() {
	for _, g := range []generators.Generator{
		&action.ActionGenerator{},
		&docs.DocsGenerator{},
		&dto.DTOGenerator{},
		&jsbridge.JSBridgeGenerator{},
		&mailer.MailerGenerator{},
		&mcp.Generator{},
		migration.New(),
		model.New(),
		&page.PageGenerator{},
		&repository.RepositoryGenerator{},
		&route.RouteGenerator{},
		&scaffold.ScaffoldGenerator{},
		&seed.SeedGenerator{},
		&service.ServiceGenerator{},
		&storage.StorageGenerator{},
		&zod.ZodGenerator{},
	} {
		generators.Register(g)
	}
}
