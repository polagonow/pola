// Package sqlite provides the SQLite GORM dialect.
package sqlite

import (
	"gorm.io/driver/sqlite"

	databasegorm "github.com/polagonow/pola/database/gorm"
)

// Dialect returns the SQLite dialect for databasegorm.WithDialect.
func Dialect() databasegorm.Dialect {
	return databasegorm.Dialect{Name: "sqlite", Open: sqlite.Open}
}
