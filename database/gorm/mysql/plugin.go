// Package mysql registers the MySQL GORM dialector.
package mysql

import (
	"gorm.io/driver/mysql"

	databasegorm "github.com/polagonow/pola/database/gorm"
)

func init() {
	databasegorm.RegisterDialector("mysql", mysql.Open)
}
