package gorm

import (
	"gorm.io/gorm"
)

// Greeting represents the greeting database table.
type Greeting struct {
	gorm.Model
	Message string `gorm:"type:varchar(255)" json:"message"`
}
