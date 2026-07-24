// Package mysql provides the MySQL driver for Ent.
package mysql

import (
	_ "github.com/go-sql-driver/mysql"

	databaseent "github.com/polagonow/pola/database/ent"
)

// Driver returns the MySQL driver for databaseent.WithDriver.
func Driver() databaseent.Driver {
	const name = "mysql"
	return databaseent.Driver{
		Name:       name,
		SQLDriver:  name,
		EntDialect: name,
	}
}
