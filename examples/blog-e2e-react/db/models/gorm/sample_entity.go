package gorm

import (
	"gorm.io/gorm"
)

// SampleEntity represents the sample_entity database table.
type SampleEntity struct {
	gorm.Model
	Name string `gorm:"type:varchar(255)" json:"name"`
}
