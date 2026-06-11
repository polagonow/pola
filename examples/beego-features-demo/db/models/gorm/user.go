package gorm

import (
	"gorm.io/gorm"
)

// User represents the user database table.
type User struct {
	gorm.Model
	Username     string `gorm:"type:varchar(255);uniqueIndex" json:"username"`
	PasswordHash string `gorm:"type:varchar(255)" json:"password_hash"`
	DisplayName  string `gorm:"type:varchar(255)" json:"display_name"`
}
