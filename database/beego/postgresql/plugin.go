// Package postgresql provides the PostgreSQL driver for Beego ORM.
package postgresql

import (
	_ "github.com/lib/pq"

	databasebeego "github.com/polagonow/pola/database/beego"
)

// Driver returns the PostgreSQL driver for databasebeego.WithDriver.
func Driver() databasebeego.Driver {
	return databasebeego.Driver{Name: "postgresql", DriverName: "postgres"}
}
