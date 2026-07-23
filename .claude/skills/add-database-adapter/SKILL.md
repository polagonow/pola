---
name: add-database-adapter
description: Add a new database adapter (SQL dialect/driver) to the Pola framework for GORM, Ent, and/or Beego. Use when asked to support a new database such as SQL Server, ClickHouse, CockroachDB, or TiDB alongside the existing sqlite, postgresql, and mysql adapters.
---

Database adapters follow the same no-init() rule as every other subsystem: each
adapter is a tiny package exporting a typed constructor — `Dialect()` for GORM,
`Driver()` for Ent/Beego — whose `Name` is the value apps put in `Polafile.hcl`'s
`database { adapter = "<name>" }`. The generated wiring passes it to the base
plugin explicitly: `databasegorm.Plugin(databasegorm.WithDialect(sqlite.Dialect()), …)`.
No global registries, no blank imports of Pola packages (blank imports of
third-party `database/sql` drivers are fine — that side effect lives upstream).

Existing adapters: `database/gorm/{sqlite,postgresql,mysql}`,
`database/ent/{sqlite,postgresql,mysql}`, and `database/beego/{sqlite,postgresql,mysql}`.

## Files to create

| File | Purpose |
|------|---------|
| `database/gorm/<name>/plugin.go` | exports `Dialect() databasegorm.Dialect` |
| `database/ent/<name>/plugin.go` | exports `Driver() databaseent.Driver` (+ blank-import of the SQL driver) |
| `database/beego/<name>/plugin.go` | exports `Driver() databasebeego.Driver` (+ blank-import of the SQL driver) |

Add whichever ORMs the database has drivers for.

---

## GORM adapter

**`database/gorm/<name>/plugin.go`** — reference: `database/gorm/sqlite/plugin.go`

```go
// Package <name> provides the <Name> GORM dialect.
package myname

import (
    myname "gorm.io/driver/<name>" // the GORM driver for this database

    databasegorm "github.com/polagonow/pola/database/gorm"
)

// Dialect returns the <Name> dialect for databasegorm.WithDialect.
func Dialect() databasegorm.Dialect {
    // Name = the Polafile `adapter` value. Open turns a DSN/URL into a Dialector.
    return databasegorm.Dialect{Name: "<name>", Open: myname.Open}
}
```

The framework resolves the adapter name from the merged Polafile config
(`database` block + `env "<env>"` override), matches it against the dialects
passed via `WithDialect` (multiple may be passed, e.g. sqlite for dev and
postgresql for prod), and calls `Open` with the connection string.

## Ent adapter

**`database/ent/<name>/plugin.go`** — reference: `database/ent/sqlite/plugin.go`

```go
// Package <name> provides the <Name> driver for Ent.
package myname

import (
    _ "github.com/<vendor>/<sql-driver>" // database/sql driver, blank import

    databaseent "github.com/polagonow/pola/database/ent"
)

// Driver returns the <Name> driver for databaseent.WithDriver.
func Driver() databaseent.Driver {
    return databaseent.Driver{
        Name:       "<name>",
        SQLDriver:  "<database/sql driver name>", // e.g. "sqlite3", "pgx"
        EntDialect: "<ent dialect>",              // e.g. "sqlite3", "postgres"
    }
}
```

## Beego adapter

**`database/beego/<name>/plugin.go`** — reference: `database/beego/sqlite/plugin.go`.
Same shape as Ent with `databasebeego.Driver{Name, DriverName}`, where
`DriverName` is the `database/sql` driver name Beego ORM registers against.

## Wire into the CLI

- Add `<name>` to the accepted adapter values wherever `sqlite | postgresql | mysql`
  are enumerated: `polafile/` (adapter validation/merge helpers).
- The generated `pola_plugins.go` wiring comes from
  `internal/autoload/pluginimports`: the template imports the adapter package as
  `databaseadapter "…/database/<orm>/<adapter>"` and `GenerateSource` computes the
  option expression (`WithDialect(databaseadapter.Dialect())` for gorm,
  `WithDriver(databaseadapter.Driver())` for ent/beego). Adapter packages that
  follow the naming convention above need no template changes.
- Connection-string building: check `polafile.DatabaseURL` / the adapter switch in
  `database/dsn` for how host/port/user/name become a DSN, and add a case for the
  new adapter's DSN format.
- ⚠️ `pola db migrate/rollback/status/reset/schema:load` (the Atlas-backed runner in
  `internal/cli/db.go`) is **sqlite-only** today. A new adapter works at app runtime,
  but the migration runner will reject it — extend `internal/cli/db.go` too if the
  database should be migratable via the CLI, or document that migrations must be
  applied externally.

## Verify

```
go build ./...
go test ./database/... ./internal/autoload/pluginimports/
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
