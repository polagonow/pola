// Package sqlite registers the SQLite GORM dialector.
package sqlite

import (
	"gorm.io/driver/sqlite"

	databasegorm "github.com/polagonow/pola/database/gorm"
)

func init() {
	databasegorm.RegisterDialector("sqlite", sqlite.Open)
}
