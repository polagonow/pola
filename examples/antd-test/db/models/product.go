package models

import "time"

type Product struct {
	ID        uint      `pola:"pk" json:"id"`
	Name      string    `pola:"type:string" json:"name" validate:"required"`
	Amount    int       `pola:"type:int" json:"amount" validate:"required"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Product) TableName() string { return "products" }
