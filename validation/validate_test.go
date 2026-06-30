package validation

import "testing"

func TestValidate_Valid(t *testing.T) {
	v := struct {
		Name string `validate:"required"`
	}{Name: "alice"}

	if err := Validate(v); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	v := struct {
		Name string `validate:"required"`
	}{}

	if err := Validate(v); err == nil {
		t.Fatal("expected validation error for empty required field")
	}
}

func TestValidate_Optional(t *testing.T) {
	v := struct {
		Name string `validate:"omitempty,alpha"`
	}{}

	if err := Validate(v); err != nil {
		t.Fatalf("expected no error for optional field, got %v", err)
	}
}

func TestValidate_NoTags(t *testing.T) {
	v := struct {
		Name string
	}{}

	if err := Validate(v); err != nil {
		t.Fatalf("expected no error for untagged struct, got %v", err)
	}
}
