package models

import "time"

type Article struct {
	ID        uint      `pola:"pk" json:"id"`
	Title     string    `pola:"type:string" json:"title" validate:"required"`
	Body      string    `pola:"type:string" json:"body" validate:"required"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Article) TableName() string { return "articles" }
