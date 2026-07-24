// Package sqlite provides the SQLite driver for Ent.
package sqlite

import (
	_ "github.com/mattn/go-sqlite3"

	databaseent "github.com/polagonow/pola/database/ent"
)

// Driver returns the SQLite driver for databaseent.WithDriver.
func Driver() databaseent.Driver {
	return databaseent.Driver{
		Name:       "sqlite",
		SQLDriver:  "sqlite3",
		EntDialect: "sqlite3",
	}
}
