// Package postgresql registers the PostgreSQL GORM dialector.
package postgresql

import (
	"gorm.io/driver/postgres"

	databasegorm "github.com/polagonow/pola/database/gorm"
)

func init() {
	databasegorm.RegisterDialector("postgresql", postgres.Open)
}
