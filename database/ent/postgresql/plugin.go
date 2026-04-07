// Package postgresql registers the PostgreSQL driver for Ent.
package postgresql

import (
	_ "github.com/lib/pq"

	databaseent "github.com/polagonow/pola/database/ent"
)

func init() {
	databaseent.RegisterDriver("postgresql", databaseent.DriverInfo{
		SQLDriver:  "postgres",
		EntDialect: "postgres",
	})
}
