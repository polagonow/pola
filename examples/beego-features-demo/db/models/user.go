package models

import "time"

type User struct {
	ID           uint      `pola:"pk" json:"id"`
	Username     string    `pola:"type:string" json:"username"`
	PasswordHash string    `pola:"type:string" json:"-"`
	DisplayName  string    `pola:"type:string" json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }
