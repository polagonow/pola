// Package postgresql registers the PostgreSQL driver for Beego ORM.
package postgresql

import (
	_ "github.com/lib/pq"

	databasebeego "github.com/polagonow/pola/database/beego"
)

func init() {
	databasebeego.RegisterDriver("postgresql", "postgres")
}
