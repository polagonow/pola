package gorm

import (
	"gorm.io/gorm"
)

// Todo represents the todo database table.
type Todo struct {
	gorm.Model
	Title     string `gorm:"type:varchar(255)" json:"title"`
	Completed bool   `json:"completed"`
}
