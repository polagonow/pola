# polafile

The `polafile` package reads and writes `Polafile.hcl` — the project-level configuration file that locks user choices made during `pola new`.

Generators, migration managers, and other CLI commands read this file to pick up the user's preferences automatically.

## Polafile.hcl

A `Polafile.hcl` is generated at the project root when running `pola new`. It uses [HashiCorp HCL](https://github.com/hashicorp/hcl) syntax.

```hcl
pola {
  package         = "blog-e2e-react"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  package_manager = "pnpm"
  app             = "app"
  actions         = "actions"
  routes          = "routes"

  csrf {
    enabled = true
  }

  security_headers {
    enabled = true
  }

  cache {
    enabled = true
    adapter = "memory"

    env "production" {
      adapter  = "redis"
      host     = "localhost"
      port     = "6379"
    }
  }

  database {
    models = "models"
    orm    = "ent"

    migrations {
      directory = "migrations"
      format    = "sql"
    }

    env "development" {
      adapter = "sqlite"
    }

    env "production" {
      adapter = "postgresql"
    }
  }
}
```

### Top-level attributes

All attributes are optional.

| Attribute         | Description                              | Example values                                |
|-------------------|------------------------------------------|-----------------------------------------------|
| `package`         | App's Go module name (set by `pola new`)  | `blog-e2e-react`, `my-app`                   |
| `version`         | Pola CLI version used to create the project | `0.1.0`                                     |
| `renderer`        | View renderer                            | `react`                                       |
| `engine`          | JavaScript engine                        | `goja`                                        |
| `bundler`         | JS bundler                               | `esbuild`                                     |
| `router`          | Router style                             | `nextjs`                                      |
| `css`             | CSS processor                            | `tailwind`, `none`                            |
| `package_manager` | JS package manager                       | `pnpm`, `npm`, `yarn`                         |
| `app`             | Directory for frontend app               | `app`                                         |
| `actions`         | Directory for server actions             | `actions`                                     |
| `routes`          | Directory for API routes                 | `routes`                                      |

### Nested blocks

#### `csrf`

| Field     | Type | Default | Description              |
|-----------|------|---------|--------------------------|
| `enabled` | bool | `true`  | Enable CSRF protection   |

Supports per-environment overrides via `env` sub-blocks.

#### `security_headers`

| Field     | Type | Default | Description                |
|-----------|------|---------|----------------------------|
| `enabled` | bool | `true`  | Enable security headers    |

Supports per-environment overrides via `env` sub-blocks.

#### `cache`

| Field      | Type   | Default  | Description             |
|------------|--------|----------|-------------------------|
| `enabled`  | bool   | `false`  | Enable caching          |
| `adapter`  | string | —        | Cache backend           |
| `host`     | string | —        | Cache host              |
| `port`     | string | —        | Cache port              |
| `password` | string | —        | Cache password          |
| `db`       | string | —        | Cache database number   |

Supports per-environment overrides via `env` sub-blocks.

#### `database`

| Field     | Type   | Default | Description                       |
|-----------|--------|---------|-----------------------------------|
| `url`     | string | —       | Database connection URL           |
| `dev_url` | string | —       | Dev database URL (for migrations) |
| `models`  | string | —       | Models directory                  |
| `adapter` | string | —       | Database adapter                  |
| `orm`     | string | —       | ORM (`ent` or `gorm`)            |

Supports per-environment overrides via `env` sub-blocks.

##### `database > migrations`

| Field       | Type   | Default      | Description              |
|-------------|--------|--------------|--------------------------|
| `directory` | string | `migrations` | Migrations directory     |
| `format`    | string | `sql`        | Migration format         |

Shared across all environments (no per-env overrides).

## Resolution order

CLI commands resolve each setting in this order (first match wins):

1. Explicit CLI flag (`--bundler=webpack`)
2. Environment variable (`POLA_BUNDLER`)
3. `Polafile.hcl`
4. Hardcoded default

## Go API

```go
import "github.com/polagonow/pola/polafile"

// Load — returns nil, nil if no Polafile.hcl exists.
pf, err := polafile.Load(".")

// Save — writes Polafile.hcl to the given directory.
err := polafile.Save(".", &polafile.Polafile{
    Package:        "my-app",
    Version:        "0.1.0",
    Renderer:       "react",
    Engine:         "goja",
    Bundler:        "esbuild",
    PackageManager: "pnpm",
    CSS:            "tailwind",
    App:            "app",
    Actions:        "actions",
    Routes:         "routes",
    CSRF:           &polafile.CSRF{Enabled: true},
    SecurityHeaders: &polafile.SecurityHeaders{Enabled: true},
    Cache:          &polafile.Cache{Enabled: true, Adapter: "memory"},
    Database:       &polafile.Database{
      Models: "models",
      ORM:    "ent",
      Migrations: &polafile.Migrations{
        Directory: "migrations",
        Format:    "sql",
      },
    },
})

// PolaPackage — get the framework import path (defaults to "github.com/polagonow/pola").
pkg := pf.PolaPackage()

// CSRFEnabled — check if CSRF is enabled for an environment.
enabled := pf.CSRFEnabled("production")

// SecurityHeadersEnabled — check if security headers are enabled.
enabled := pf.SecurityHeadersEnabled("production")

// DatabaseForEnv — merge base database config with env-specific overrides.
db := pf.DatabaseForEnv("production")
fmt.Println(db.Adapter) // "postgresql"

// CacheEnabled — check if caching is enabled for an environment.
enabled := pf.CacheEnabled("production")

// CacheAdapter — get the cache adapter for an environment.
adapter := pf.CacheAdapter("production") // "redis"
```
