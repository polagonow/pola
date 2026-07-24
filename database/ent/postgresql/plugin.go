// Package postgresql provides the PostgreSQL driver for Ent.
package postgresql

import (
	_ "github.com/lib/pq"

	databaseent "github.com/polagonow/pola/database/ent"
)

// Driver returns the PostgreSQL driver for databaseent.WithDriver.
func Driver() databaseent.Driver {
	return databaseent.Driver{
		Name:       "postgresql",
		SQLDriver:  "postgres",
		EntDialect: "postgres",
	}
}
