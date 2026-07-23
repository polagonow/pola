// Package mysql provides the MySQL driver for Beego ORM.
package mysql

import (
	_ "github.com/go-sql-driver/mysql"

	databasebeego "github.com/polagonow/pola/database/beego"
)

// Driver returns the MySQL driver for databasebeego.WithDriver.
func Driver() databasebeego.Driver {
	return databasebeego.Driver{Name: "mysql", DriverName: "mysql"}
}
