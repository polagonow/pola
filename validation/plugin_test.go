package validation

import "testing"

// TestBindValidateGuard covers the validator the plugin installs: it validates
// structs against their `validate:"..."` tags and no-ops on non-structs so a
// scalar bind never yields a spurious 422.
func TestBindValidateGuard(t *testing.T) {
	type input struct {
		Name string `validate:"required"`
	}

	if err := bindValidate(&input{}); err == nil {
		t.Error("expected a validation error for a missing required field")
	}
	if err := bindValidate(&input{Name: "x"}); err != nil {
		t.Errorf("valid struct: got %v, want nil", err)
	}
	if err := bindValidate(input{Name: "x"}); err != nil {
		t.Errorf("valid non-pointer struct: got %v, want nil", err)
	}

	// Non-struct values must be ignored, not 422'd.
	n := 5
	if err := bindValidate(&n); err != nil {
		t.Errorf("pointer-to-int: got %v, want nil", err)
	}
	if err := bindValidate("scalar"); err != nil {
		t.Errorf("string: got %v, want nil", err)
	}
	if err := bindValidate(nil); err != nil {
		t.Errorf("nil: got %v, want nil", err)
	}
}
