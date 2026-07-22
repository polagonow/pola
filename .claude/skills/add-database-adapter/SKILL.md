---
name: add-database-adapter
description: Add a new database adapter (SQL dialect/driver) to the Pola framework for GORM and/or Ent. Use when asked to support a new database such as SQL Server, ClickHouse, CockroachDB, or TiDB alongside the existing sqlite, postgresql, and mysql adapters.
---

Database adapters are the one subsystem that still uses `init()` registries: each
adapter is a tiny package that registers a dialect (GORM) or driver info (Ent) under
the name apps put in `Polafile.hcl`'s `database { adapter = "<name>" }`. The
generated wiring imports the adapter package for its side effect.

Existing adapters: `database/gorm/{sqlite,postgresql,mysql}` and
`database/ent/{sqlite,postgresql,mysql}`.

## Files to create

| File | Purpose |
|------|---------|
| `database/gorm/<name>/plugin.go` | `init()` calling `databasegorm.RegisterDialector` |
| `database/ent/<name>/plugin.go` | `init()` calling `databaseent.RegisterDriver` (+ blank-import of the SQL driver) |

Add one or both depending on which ORMs the database has drivers for.

---

## GORM adapter

**`database/gorm/<name>/plugin.go`** — reference: `database/gorm/sqlite/plugin.go`

```go
// Package <name> registers the <Name> GORM dialector.
package myname

import (
    myname "gorm.io/driver/<name>" // the GORM driver for this database

    databasegorm "github.com/polagonow/pola/database/gorm"
)

func init() {
    // Key = the Polafile `adapter` value. The func turns a DSN/URL into a Dialector.
    databasegorm.RegisterDialector("<name>", myname.Open)
}
```

`RegisterDialector(adapter string, fn func(dsn string) gorm.Dialector)` — the
framework resolves the adapter name from the merged Polafile config
(`database` block + `env "<env>"` override) and calls the factory with the
connection string.

## Ent adapter

**`database/ent/<name>/plugin.go`** — reference: `database/ent/sqlite/plugin.go`

```go
// Package <name> registers the <Name> driver for Ent.
package myname

import (
    _ "github.com/<vendor>/<sql-driver>" // database/sql driver, blank import

    databaseent "github.com/polagonow/pola/database/ent"
)

func init() {
    databaseent.RegisterDriver("<name>", databaseent.DriverInfo{
        SQLDriver:  "<database/sql driver name>", // e.g. "sqlite3", "pgx"
        EntDialect: "<ent dialect>",              // e.g. "sqlite3", "postgres"
    })
}
```

## Wire into the CLI

- Add `<name>` to the accepted adapter values wherever `sqlite | postgresql | mysql`
  are enumerated: `polafile/` (adapter validation/merge helpers) and the
  `internal/autoload` template that imports `database/<orm>/<adapter>` into the
  generated `pola_plugins.go`.
- Connection-string building: check `polafile.DatabaseURL` / the adapter switch in
  `database/` for how host/port/user/name become a DSN, and add a case for the new
  adapter's DSN format.
- ⚠️ `pola db migrate/rollback/status/reset/schema:load` (the Atlas-backed runner in
  `internal/cli/db.go`) is **sqlite-only** today. A new adapter works at app runtime,
  but the migration runner will reject it — extend `internal/cli/db.go` too if the
  database should be migratable via the CLI, or document that migrations must be
  applied externally.

## Verify

```
go build ./...
go test ./database/...
```

Then run an app against the database:

```
pola new db-test --api-only -y
# Polafile.hcl:
#   database {
#     orm = "gorm"
#     env "development" { adapter = "<name>", url = "<dsn>" }
#   }
cd db-test && pola generate scaffold Item name:string && pola dev
```

CRUD through the generated repository must work end-to-end against the new
database.
