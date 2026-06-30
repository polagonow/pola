package validation

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/polagonow/pola/core"
)

type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   any    `json:"value,omitempty"`
	Message string `json:"message"`
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var sb strings.Builder
	for i, e := range ve {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", e.Field, e.Message))
	}
	return sb.String()
}

type PlaygroundValidator struct {
	v *validator.Validate
}

func New() *PlaygroundValidator {
	return &PlaygroundValidator{v: validator.New(validator.WithRequiredStructEnabled())}
}

// defaultValidator backs the package-level Validate convenience.
var defaultValidator = New()

// Validate validates v using the default go-playground validator, returning
// structured ValidationErrors on failure.
func Validate(v any) error {
	return defaultValidator.Validate(v)
}

func (pv *PlaygroundValidator) Validate(val any) error {
	err := pv.v.Struct(val)
	if err == nil {
		return nil
	}
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}
	errs := make(ValidationErrors, len(ve))
	for i, fe := range ve {
		errs[i] = ValidationError{
			Field:   fe.Field(),
			Tag:     fe.Tag(),
			Value:   fe.Value(),
			Message: fmt.Sprintf("failed on '%s' validation", fe.Tag()),
		}
	}
	return errs
}

func (pv *PlaygroundValidator) ValidateField(field any, tag string) error {
	return pv.v.Var(field, tag)
}

var _ core.Validator = (*PlaygroundValidator)(nil)
