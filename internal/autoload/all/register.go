// Package all wires all built-in autoload stages into the registry explicitly.
// Call Register once at CLI startup (cli.Execute). List order is irrelevant:
// autoload.All() sorts by Priority().
package all

import (
	"github.com/polagonow/pola/internal/autoload"
	"github.com/polagonow/pola/internal/autoload/actionbridge"
	"github.com/polagonow/pola/internal/autoload/dbembed"
	"github.com/polagonow/pola/internal/autoload/dbseed"
	"github.com/polagonow/pola/internal/autoload/embed"
	"github.com/polagonow/pola/internal/autoload/entclient"
	"github.com/polagonow/pola/internal/autoload/mcp"
	"github.com/polagonow/pola/internal/autoload/pluginimports"
	"github.com/polagonow/pola/internal/autoload/repos"
	"github.com/polagonow/pola/internal/autoload/routes"
	"github.com/polagonow/pola/internal/autoload/services"
)

// Register adds every built-in autoload stage to the registry.
func Register() {
	for _, a := range []autoload.Autoload{
		actionbridge.New(),
		dbembed.New(),
		dbseed.New(),
		embed.New(),
		entclient.New(),
		mcp.New(),
		pluginimports.New(),
		repos.New(),
		routes.New(),
		services.New(),
	} {
		autoload.Register(a)
	}
}
