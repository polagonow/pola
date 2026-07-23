// Package postgresql provides the PostgreSQL GORM dialect.
package postgresql

import (
	"gorm.io/driver/postgres"

	databasegorm "github.com/polagonow/pola/database/gorm"
)

// Dialect returns the PostgreSQL dialect for databasegorm.WithDialect.
func Dialect() databasegorm.Dialect {
	return databasegorm.Dialect{Name: "postgresql", Open: postgres.Open}
}
