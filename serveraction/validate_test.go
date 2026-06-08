package serveraction

import (
	"math"
	"net/http/httptest"
	"testing"
)

func TestValidateAndSanitizeArgs_StripsDangerousKeys(t *testing.T) {
	args := []any{
		map[string]any{
			"safe":      "ok",
			"__proto__": map[string]any{"polluted": true},
			"nested": map[string]any{
				"constructor": "bad",
				"keep":        1.0,
			},
		},
	}
	out, err := ValidateAndSanitizeArgs(args, ProductionConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := out[0].(map[string]any)
	if _, ok := m["__proto__"]; ok {
		t.Error("__proto__ should be stripped")
	}
	if m["safe"] != "ok" {
		t.Error("safe key dropped")
	}
	nested := m["nested"].(map[string]any)
	if _, ok := nested["constructor"]; ok {
		t.Error("nested constructor should be stripped")
	}
	if nested["keep"] != 1.0 {
		t.Error("nested keep dropped")
	}
}

func TestValidateAndSanitizeArgs_Limits(t *testing.T) {
	cfg := ProductionConfig()

	// depth
	deep := any("leaf")
	for i := 0; i < cfg.MaxDepth+2; i++ {
		deep = []any{deep}
	}
	if _, err := ValidateAndSanitizeArgs([]any{deep}, cfg); err == nil {
		t.Error("expected depth error")
	}

	// string length
	long := make([]byte, cfg.MaxStringLength+1)
	if _, err := ValidateAndSanitizeArgs([]any{string(long)}, cfg); err == nil {
		t.Error("expected string-length error")
	}

	// array length
	big := make([]any, cfg.MaxArrayLength+1)
	if _, err := ValidateAndSanitizeArgs([]any{big}, cfg); err == nil {
		t.Error("expected array-length error")
	}

	// special numbers
	if _, err := ValidateAndSanitizeArgs([]any{math.Inf(1)}, cfg); err == nil {
		t.Error("expected Infinity error")
	}
	if _, err := ValidateAndSanitizeArgs([]any{math.NaN()}, cfg); err == nil {
		t.Error("expected NaN error")
	}
}

func TestIsReservedExportName(t *testing.T) {
	for _, n := range []string{"then", "catch", "finally", "constructor", "Symbol"} {
		if !IsReservedExportName(n) {
			t.Errorf("%q should be reserved", n)
		}
	}
	for _, n := range []string{"addTodo", "getPosts", "default"} {
		if IsReservedExportName(n) {
			t.Errorf("%q should not be reserved", n)
		}
	}
}

func TestCheckOrigin(t *testing.T) {
	t.Run("matching origin", func(t *testing.T) {
		r := httptest.NewRequest("POST", "http://example.com/_pola/action", nil)
		r.Host = "example.com"
		r.Header.Set("Origin", "http://example.com")
		if err := CheckOrigin(r); err != nil {
			t.Errorf("expected ok, got %v", err)
		}
	})
	t.Run("mismatched origin", func(t *testing.T) {
		r := httptest.NewRequest("POST", "http://example.com/_pola/action", nil)
		r.Host = "example.com"
		r.Header.Set("Origin", "http://evil.com")
		if err := CheckOrigin(r); err == nil {
			t.Error("expected forbidden")
		}
	})
	t.Run("referer fallback", func(t *testing.T) {
		r := httptest.NewRequest("POST", "http://example.com/_pola/action", nil)
		r.Host = "example.com"
		r.Header.Set("Referer", "http://example.com/page")
		if err := CheckOrigin(r); err != nil {
			t.Errorf("expected ok, got %v", err)
		}
	})
	t.Run("no headers", func(t *testing.T) {
		r := httptest.NewRequest("POST", "http://example.com/_pola/action", nil)
		r.Host = "example.com"
		if err := CheckOrigin(r); err == nil {
			t.Error("expected forbidden when no origin/referer")
		}
	})
}
