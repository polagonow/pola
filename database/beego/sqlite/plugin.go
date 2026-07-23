// Package sqlite provides the SQLite driver for Beego ORM.
package sqlite

import (
	_ "github.com/mattn/go-sqlite3"

	databasebeego "github.com/polagonow/pola/database/beego"
)

// Driver returns the SQLite driver for databasebeego.WithDriver.
func Driver() databasebeego.Driver {
	return databasebeego.Driver{Name: "sqlite", DriverName: "sqlite3"}
}
