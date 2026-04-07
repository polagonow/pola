// Package sqlite registers the SQLite driver for Ent.
package sqlite

import (
	_ "github.com/mattn/go-sqlite3"

	databaseent "github.com/polagonow/pola/database/ent"
)

func init() {
	databaseent.RegisterDriver("sqlite", databaseent.DriverInfo{
		SQLDriver:  "sqlite3",
		EntDialect: "sqlite3",
	})
}
