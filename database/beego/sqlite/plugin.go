// Package sqlite registers the SQLite driver for Beego ORM.
package sqlite

import (
	_ "github.com/mattn/go-sqlite3"

	databasebeego "github.com/polagonow/pola/database/beego"
)

func init() {
	databasebeego.RegisterDriver("sqlite", "sqlite3")
}
