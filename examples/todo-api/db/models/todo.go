package models

import "time"

// Todo is the canonical Todo model — single source of truth.
type Todo struct {
	ID        uint      `pola:"pk" json:"id"`
	Title     string    `pola:"type:string" json:"title" validate:"required"`
	Completed bool      `pola:"type:bool" json:"completed" validate:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Todo) TableName() string { return "todos" }
