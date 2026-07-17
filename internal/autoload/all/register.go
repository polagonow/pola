// Package all registers all built-in autoloads.
// Import this package with a blank import to trigger registration.
package all

import (
	_ "github.com/polagonow/pola/internal/autoload/actionbridge"
	_ "github.com/polagonow/pola/internal/autoload/dbembed"
	_ "github.com/polagonow/pola/internal/autoload/dbseed"
	_ "github.com/polagonow/pola/internal/autoload/embed"
	_ "github.com/polagonow/pola/internal/autoload/entclient"
	_ "github.com/polagonow/pola/internal/autoload/mcp"
	_ "github.com/polagonow/pola/internal/autoload/pluginimports"
	_ "github.com/polagonow/pola/internal/autoload/repos"
	_ "github.com/polagonow/pola/internal/autoload/routes"
	_ "github.com/polagonow/pola/internal/autoload/services"
)
