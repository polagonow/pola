// Package mysql registers the MySQL driver for Beego ORM.
package mysql

import (
	_ "github.com/go-sql-driver/mysql"

	databasebeego "github.com/polagonow/pola/database/beego"
)

func init() {
	databasebeego.RegisterDriver("mysql", "mysql")
}
