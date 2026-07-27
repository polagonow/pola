# Running the examples with Docker

Every example ships a `Dockerfile` that produces a **single static binary served
from a `scratch` image** — no OS, no Node.js at runtime, just the compiled Pola
app (front-end assets are embedded during the build). Database-backed examples
also ship a `docker-compose.yml` that runs the app against PostgreSQL.

## Build context is the repository root

Examples resolve the framework through a local replace directive:

```
replace github.com/polagonow/pola => ../..
```

so the Docker build **context must be the repo root**, and the Dockerfile is
selected with `-f`:

```bash
# from the repository root
docker build -f examples/<name>/Dockerfile -t <name> .
docker run --rm -p 3000:3000 <name>
```

The app listens on `PORT` (default `3000`); set `POLA_ADDRESS` to change the
bind host.

## How the image is built

There are three Dockerfile shapes:

| Type | Examples | Build |
|---|---|---|
| **Web** (has `web/`) | antd-test, beego-features-demo, blog-e2e-react, fumadocs-docs, image-demo, mcp-hello, saas-starter, server-actions-demo, slds-test | builds the `pola` CLI, then `pola build` (Node + pnpm bundle the `.tsx` front-end with esbuild, assets embedded) |
| **API-only** (Polafile, no `web/`) | beego-demo, ent-demo, features-showcase, gorm-demo, my-api, todo-api | builds the `pola` CLI, then `pola build` (plugin wiring, no bundling) |
| **Plain** (no Polafile) | env-config-demo, validation | plain `CGO_ENABLED=0 go build` |

All images compile with `CGO_ENABLED=0` so the binary is fully static and runs
on `scratch`.

**Building never requires a database connection.** The database driver connects
lazily on the first query, so `pola build` (including the web bundle step, which
runs the app to emit assets) wires the app without dialing a database.

## Database-backed examples run on PostgreSQL

A `scratch` image cannot carry the cgo SQLite driver (`mattn/go-sqlite3`), so
every database-backed example is built with the **pure-Go PostgreSQL adapter**
and ships a `docker-compose.yml` that starts Postgres alongside the app:

`antd-test`, `blog-e2e-react`, `slds-test`, `saas-starter`, `beego-features-demo`,
`mcp-hello`, `todo-api`, `features-showcase`, `gorm-demo`, `beego-demo`, `ent-demo`.

```bash
# from the repository root
docker compose -f examples/antd-test/docker-compose.yml up --build
```

SQLite remains the default for local development (`pola dev`); the Polafiles are
unchanged. Only the Docker build forces the Postgres adapter, via
`POLA_DATABASE_ADAPTER=postgresql` (the four examples that already target
Postgres in their `production` environment need no override).

Connection settings are passed to the binary at runtime through the `DATABASE_*`
environment variables it reads on boot:

| Variable | Purpose |
|---|---|
| `DATABASE_ADAPTER` | `postgresql` |
| `DATABASE_HOST` / `DATABASE_PORT` | Postgres address (`db:5432` in compose) |
| `DATABASE_USER` / `DATABASE_PASSWORD` | credentials |
| `DATABASE_NAME` | database name |
| `DATABASE_SSLMODE` | `disable` for the local container |
| `DATABASE_URL` | full DSN; overrides the individual fields |

`saas-starter` additionally reads `AUTH_SECRET`, `BASE_URL`,
`STRIPE_SECRET_KEY`, and `STRIPE_WEBHOOK_SECRET` (see its `.env.example`); the
compose file wires dev defaults you should override.

### ⚠️ Migrations

The committed migrations under `db/migrations/` are **SQLite-flavoured** (they
were generated for the dev database) and will not apply to Postgres as-is. The
app boots and connects regardless; generate Postgres migrations before relying
on the data layer, e.g.:

```bash
POLA_ENV=production pola generate migration <name>   # against a Postgres target
```
