# `Polafile.hcl` reference

`Polafile.hcl` is generated at the project root by `pola new` and is the source of truth for every
`pola` command. It uses [HCL](https://github.com/hashicorp/hcl). **All configuration nests inside a
single top-level `pola { }` block.** Every attribute and block is optional.

## Resolution order

For any setting, the CLI resolves **first match wins**:

1. Explicit CLI flag (e.g. `--bundler=esbuild`)
2. Environment variable (e.g. `POLA_BUNDLER`)
3. `Polafile.hcl`
4. Hardcoded default

## Top-level attributes

| Attribute | Purpose | Example |
|-----------|---------|---------|
| `package` | Go module name (set by `pola new`) | `"myapp"` |
| `version` | Pola CLI version that created the project | `"0.1.0"` |
| `renderer` | View renderer | `"react"` |
| `engine` | **JS engine** (CLI flag is `--vm`, but the key here is `engine`) | `"goja"` |
| `bundler` | JS bundler | `"esbuild"` |
| `router` | Router style | `"nextjs"` |
| `css` | CSS processor | `"tailwind"`, `"sass"`, `"none"` |
| `ui` | UI library | `"shadcn"`, `"antd"`, `"none"` |
| `package_manager` | JS package manager | `"pnpm"`, `"npm"`, `"yarn"` |
| `app` | Frontend app directory | `"web"` |
| `actions` | Server actions directory | `"actions"` |
| `routes` | API routes directory | `"routes"` |
| `repositories` | Repositories directory | `"repositories"` |
| `services` | Services directory | `"services"` |

## Blocks

Every block listed below may contain one or more `env "<name>" { … }` sub-blocks that override the
block's attributes for that environment (selected via `--env`, default `development`).

### `csrf` / `security_headers`
```hcl
csrf             { enabled = true }
security_headers { enabled = true }
```
Both support `env "<name>" { enabled = false }`.

### `cache`
Attributes: `enabled`, `adapter` (`memory` | `redis`), `host`, `port`, `password`, `db`.
```hcl
cache {
  enabled = true
  adapter = "memory"

  env "production" {
    adapter  = "redis"
    host     = "localhost"
    port     = "6379"
    password = ""
    db       = "0"
  }
}
```

### `database`
Attributes: `url`, `host`, `port`, `user`, `password`, `name`, `models`, `adapter`
(`sqlite` | `postgresql` | `mysql`), `orm` (`gorm` | `ent`), `orm_implementations`.
Nested `migrations { directory, format, dev_url }`. `env` overrides accept the same connection
attributes.
```hcl
database {
  models = "db/models"
  orm    = "gorm"

  migrations {
    directory = "db/migrations"
    format    = "hcl"          # hcl | sql
    dev_url   = "sqlite://file?mode=memory"
  }

  env "development" { adapter = "sqlite" }
  env "production"  { adapter = "postgresql", url = "postgres://user:pass@host/db" }
}
```
The generators read `database.orm` to decide which ORM templates to emit. Remember: `pola db …`
(the migration runner) is **sqlite-only** regardless of these settings.

### `storage`
Attributes: `driver` (`fs` | `rclone`), `root`, `config_path`. Configured by `pola generate storage`.
```hcl
storage {
  driver = "fs"
  root   = "uploads"

  env "production" {
    driver      = "rclone"
    root        = "myremote:bucket/uploads"
    config_path = "/etc/rclone/rclone.conf"
  }
}
```

### `mailer`
Attributes: `renderer`, `transport`, `from`, `host`, `port`, `username`, `password`, `tls`.
```hcl
mailer {
  renderer  = "react"
  transport = "smtp"
  from      = "noreply@example.com"
  host      = "smtp.example.com"
  port      = "587"
  username  = "user"
  password  = "pass"
  tls       = "starttls"
}
```

### `image_processing`
Attributes: `enabled`, `adapter` (`imaging`), `path` (HTTP prefix, default `/_pola/image`),
`max_width`, `max_height`, `format`.
```hcl
image_processing {
  enabled    = true
  adapter    = "imaging"
  path       = "/_pola/image"
  max_width  = 4096
  max_height = 4096
  format     = "jpeg"
}
```

### `mcp`
Attributes: `enabled`, `transport` (`http` streamable | `sse` legacy | `stdio`), `mount`,
`name`, `version`, `instructions`. Add it with `pola generate mcp init`.
```hcl
mcp {
  enabled   = true
  transport = "http"
  mount     = "/mcp"
  name      = "myapp"
  version   = "0.1.0"

  env "production" { transport = "http" }
}
```

### `testing`
Attributes: `generate_tests` (bool), `framework` (`vitest` | `jest`).
```hcl
testing {
  generate_tests = true
  framework      = "vitest"
}
```

## Environment variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `PORT` | Dev server port | `3000` |
| `POLA_RENDERER` | View renderer | `react` |
| `POLA_BUNDLER` | JS bundler | `esbuild` |
| `POLA_ROUTER` | Router style | `nextjs` |
| `POLA_CSS` | CSS processor | `tailwind` |
| `POLA_VM` | JS engine | `goja` |
| `POLA_CACHE` | Cache adapter | `memory` |
| `POLA_CSRF` | Enable CSRF (`false` disables) | `true` |
| `POLA_SECURITY_HEADERS` | Enable security headers | `true` |
| `POLA_IMAGE_PROCESSING` | Image adapter (`imaging`, `none`) | — |
| `POLA_IMAGE_PROCESSING_PATH` | Image endpoint prefix | `/_pola/image` |
| `POLA_IMAGE_PROCESSING_MAX_WIDTH` / `_MAX_HEIGHT` | Output clamps | `4096` |
| `POLA_IMAGE_PROCESSING_FORMAT` | Default output format | `jpeg` |
| `POLA_WEBAPP_PATH` | Web app dir | `./web` |
| `POLA_PM` | JS package manager | autodetect |
| `POLA_ENV` | Environment label exposed to runtime | `development` (in `pola dev`) |
| `POLA_BUILD_ONLY` | Set by `pola build` stage 1 (bundle & exit) | — |
| `POLA_PUBLIC_DIR` | Output dir for the bundle stage | `./public` |
| `CGO_ENABLED` | Forwarded to `go build` | `1` |

## Two complete examples

**Dev-sqlite / prod-postgres app:**
```hcl
pola {
  package         = "myapp"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"

  csrf             { enabled = true }
  security_headers { enabled = true }
  cache            { enabled = true, adapter = "memory" }

  database {
    models = "db/models"
    orm    = "gorm"
    migrations { directory = "db/migrations", format = "hcl" }
    env "development" { adapter = "sqlite" }
    env "production"  { adapter = "postgresql" }
  }
}
```

**MCP-enabled app:**
```hcl
pola {
  package         = "mcp-hello"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  ui              = "none"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"
  repositories    = "repositories"
  services        = "services"

  csrf             { enabled = false }
  security_headers { enabled = false }
  cache            { enabled = true, adapter = "memory" }

  database {
    models  = "db/models"
    orm     = "gorm"
    adapter = "sqlite"
    migrations { directory = "db/migrations", format = "hcl" }
  }

  mcp {
    enabled   = true
    transport = "http"
    mount     = "/mcp"
  }
}
```
