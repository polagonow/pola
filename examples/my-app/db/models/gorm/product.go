package gorm

import (
	"gorm.io/gorm"
)

// Product represents the product database table.
type Product struct {
	gorm.Model
	Name  string  `gorm:"type:varchar(255)" json:"name"`
	Price float64 `json:"price"`
}
