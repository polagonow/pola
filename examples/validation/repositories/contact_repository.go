package repositories

import (
	"github.com/polagonow/pola/repository"
)

// Contact represents a contact entity showcasing govalidator field types.
//
// Equivalent CLI definition:
//
//	pola generate repository Contact name:alpha email:email website:url? phone:numeric?
type Contact struct {
	ID      uint   `json:"id"`
	Name    string `json:"name" validate:"required,alpha"`
	Email   string `json:"email" validate:"required,email"`
	Website string `json:"website" validate:"omitempty,url"`
	Phone   string `json:"phone" validate:"omitempty,numeric"`
}

// ContactRepository defines the contract for contact persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type ContactRepository interface {
	repository.Repository[Contact, uint]
}
