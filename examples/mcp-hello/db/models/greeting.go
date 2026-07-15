package models

import "time"

type Greeting struct {
	ID        uint      `pola:"pk" json:"id"`
	Message   string    `pola:"type:string" json:"message" validate:"required"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Greeting) TableName() string { return "greetings" }
