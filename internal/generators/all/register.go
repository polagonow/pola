// Package all registers all built-in generators.
// Import this package with a blank import to trigger registration.
package all

import (
	_ "github.com/polagonow/pola/internal/generators/action"
	_ "github.com/polagonow/pola/internal/generators/jsbridge"
	_ "github.com/polagonow/pola/internal/generators/mailer"
	_ "github.com/polagonow/pola/internal/generators/mcp"
	_ "github.com/polagonow/pola/internal/generators/migration"
	_ "github.com/polagonow/pola/internal/generators/model"
	_ "github.com/polagonow/pola/internal/generators/page"
	_ "github.com/polagonow/pola/internal/generators/repository"
	_ "github.com/polagonow/pola/internal/generators/route"
	_ "github.com/polagonow/pola/internal/generators/scaffold"
	_ "github.com/polagonow/pola/internal/generators/service"
	_ "github.com/polagonow/pola/internal/generators/storage"
	_ "github.com/polagonow/pola/internal/generators/zod"
)
