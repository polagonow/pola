package repositories

import (
	"context"
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

// ContactRepository defines the contract for contact persistence operations.
type ContactRepository interface {
	Create(ctx context.Context, entity *Contact) error
	Get(ctx context.Context, id uint) (*Contact, error)
	List(ctx context.Context, params ListParams) (*ListResult[*Contact], error)
	Update(ctx context.Context, entity *Contact) error
	Delete(ctx context.Context, id uint) error
}
