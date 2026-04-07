// Package mysql registers the MySQL driver for Ent.
package mysql

import (
	_ "github.com/go-sql-driver/mysql"

	databaseent "github.com/polagonow/pola/database/ent"
)

func init() {
	databaseent.RegisterDriver("mysql", databaseent.DriverInfo{
		SQLDriver:  "mysql",
		EntDialect: "mysql",
	})
}
