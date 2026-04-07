package routes

import govalidator "github.com/asaskevich/govalidator"

// Validate runs struct validation using govalidator tags.
// Returns nil if all fields pass validation.
func Validate(v any) error {
	_, err := govalidator.ValidateStruct(v)
	return err
}
