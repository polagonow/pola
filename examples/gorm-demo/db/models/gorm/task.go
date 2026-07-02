package gorm

import (
	"gorm.io/gorm"
)

// Task represents the task database table.
type Task struct {
	gorm.Model
	Title string `gorm:"type:varchar(255)" json:"title"`
	Done  bool   `json:"done"`
}
