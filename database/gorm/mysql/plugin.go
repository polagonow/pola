// Package mysql provides the MySQL GORM dialect.
package mysql

import (
	"gorm.io/driver/mysql"

	databasegorm "github.com/polagonow/pola/database/gorm"
)

// Dialect returns the MySQL dialect for databasegorm.WithDialect.
func Dialect() databasegorm.Dialect {
	return databasegorm.Dialect{Name: "mysql", Open: mysql.Open}
}
