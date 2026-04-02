package gorm

import (
	"gorm.io/gorm"
)

// Text represents the text database table.
type Text struct {
	gorm.Model
	Name string `gorm:"type:varchar(255)" json:"name"`
	Password string `gorm:"type:varchar(255)" json:"password"`
	Age int `json:"age"`
}
