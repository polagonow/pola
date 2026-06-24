package env

import "testing"

func TestString(t *testing.T) {
	t.Setenv("TEST_STR", "hello")
	if got := String("TEST_STR", "default"); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
	if got := String("TEST_STR_MISSING", "default"); got != "default" {
		t.Fatalf("expected default, got %q", got)
	}
}

func TestBool(t *testing.T) {
	t.Setenv("TEST_BOOL_TRUE", "true")
	t.Setenv("TEST_BOOL_ONE", "1")
	t.Setenv("TEST_BOOL_FALSE", "false")

	if !Bool("TEST_BOOL_TRUE", false) {
		t.Fatal("expected true")
	}
	if !Bool("TEST_BOOL_ONE", false) {
		t.Fatal("expected true for 1")
	}
	if Bool("TEST_BOOL_FALSE", true) {
		t.Fatal("expected false")
	}
	if !Bool("TEST_BOOL_MISSING", true) {
		t.Fatal("expected fallback true")
	}
}

func TestInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	t.Setenv("TEST_INT_BAD", "abc")

	if got := Int("TEST_INT", 0); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
	if got := Int("TEST_INT_BAD", 10); got != 10 {
		t.Fatalf("expected fallback 10, got %d", got)
	}
	if got := Int("TEST_INT_MISSING", 99); got != 99 {
		t.Fatalf("expected fallback 99, got %d", got)
	}
}

func TestFloat64(t *testing.T) {
	t.Setenv("TEST_FLOAT", "3.14")
	t.Setenv("TEST_FLOAT_BAD", "xyz")

	if got := Float64("TEST_FLOAT", 0); got != 3.14 {
		t.Fatalf("expected 3.14, got %f", got)
	}
	if got := Float64("TEST_FLOAT_BAD", 1.5); got != 1.5 {
		t.Fatalf("expected fallback 1.5, got %f", got)
	}
	if got := Float64("TEST_FLOAT_MISSING", 2.0); got != 2.0 {
		t.Fatalf("expected fallback 2.0, got %f", got)
	}
}
