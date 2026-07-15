package models

import "time"

type SampleEntity struct {
	ID        uint      `pola:"pk" json:"id"`
	Name      string    `pola:"type:string" json:"name" validate:"required"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SampleEntity) TableName() string { return "sample_entities" }
