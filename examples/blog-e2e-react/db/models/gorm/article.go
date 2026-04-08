package gorm

import (
	"gorm.io/gorm"
)

// Article represents the article database table.
type Article struct {
	gorm.Model
	Title string `gorm:"type:varchar(255)" json:"title"`
	Body  string `gorm:"type:text" json:"body"`
}
