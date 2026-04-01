# polafile

The `polafile` package reads and writes `Polafile.hcl` — the project-level configuration file that locks user choices made during `pola new`.

Generators, migration managers, and other CLI commands read this file to pick up the user's preferences automatically.

## Polafile.hcl

A `Polafile.hcl` is generated at the project root when running `pola new`. It uses [HashiCorp HCL](https://github.com/hashicorp/hcl) syntax.

```hcl
pola {
  package         = "github.com/polagonow/pola"
  version         = "0.1.0"
  renderer        = "react@^19.0.0"
  engine          = "goja@0.0.0-20240220"
  bundler         = "esbuild@^0.21.0"
  router          = "nextjs"
  css             = "tailwind@^4.0.0"
  cache           = "memory"
  package_manager = "pnpm@^9.0.0"
  app_dir         = "app"
  actions_dir     = "actions"
  routes_dir      = "routes"

  development {
    cache   = "memory"
    bundler = "esbuild@^0.21.0"
  }

  production {
    cache   = "redis"
    bundler = "webpack@^5.0.0"
    css     = "postcss@^8.0.0"
  }
}
```

### Attributes

Versioned attributes use `name@version` format (like package.json). The version part is optional — `"react"` and `"react@^19.0.0"` are both valid.

| Attribute         | Description                | Example values                          |
|-------------------|----------------------------|-----------------------------------------|
| `package`         | Pola framework Go import path | `github.com/polagonow/pola` (default) |
| `version`         | Pola CLI version used to create the project | `0.1.0`                    |
| `renderer`        | View renderer              | `react@^19.0.0`                         |
| `engine`          | JavaScript engine          | `goja@0.0.0-20240220`                   |
| `bundler`         | JS bundler                 | `esbuild@^0.21.0`, `webpack@^5.0.0`    |
| `router`          | Router style               | `nextjs`                                |
| `css`             | CSS processor              | `tailwind@^4.0.0`, `postcss@^8.0.0`, `none` |
| `cache`           | Cache backend              | `memory`, `redis`                       |
| `package_manager` | JS package manager         | `pnpm@^9.0.0`, `npm@^10.0.0`, `yarn@^4.0.0` |
| `app_dir`         | Directory for frontend app | `app`                                   |
| `actions_dir`     | Directory for server actions | `actions`                             |
| `routes_dir`      | Directory for API routes   | `routes`                                |

All attributes are optional.

### Environment blocks

The `development` and `production` blocks override base attributes for that environment. Values not set in an environment block inherit from the parent `pola` block.

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
    Version:        "0.1.0",
    Renderer:       "react@^19.0.0",
    Engine:         "goja@0.0.0-20240220",
    Bundler:        "esbuild@^0.21.0",
    PackageManager: "pnpm@^9.0.0",
    CSS:            "tailwind@^4.0.0",
    Cache:          "memory",
    AppDir:         "app",
    ActionsDir:     "actions",
    RoutesDir:      "routes",
})

// PolaPackage — get the framework import path (defaults to "github.com/polagonow/pola").
pkg := pf.PolaPackage()

// ForEnv — merge environment overrides on top of base values.
prod := pf.ForEnv("production")
fmt.Println(prod.Bundler)  // "webpack@^5.0.0" (from production block)
fmt.Println(prod.Renderer) // "react@^19.0.0" (inherited from base)

// ParseVersioned — split "name@version" into parts.
name, ver := polafile.ParseVersioned(prod.CSS) // "postcss", "^8.0.0"

// FormatVersioned — join name and version back together.
s := polafile.FormatVersioned("tailwind", "^4.0.0") // "tailwind@^4.0.0"
```
