package models

import "time"

// Task is the canonical Task model — single source of truth.
type Task struct {
	ID        uint      `pola:"pk" json:"id"`
	Title     string    `pola:"type:string" json:"title" validate:"required"`
	Done      bool      `pola:"type:bool" json:"done" validate:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Task) TableName() string { return "tasks" }
