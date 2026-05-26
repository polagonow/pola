package prometheus

import (
	"testing"
	"time"
)

func TestRecordRequest_NoPanic(t *testing.T) {
	m := New()
	// Should not panic with normal inputs.
	m.RecordRequest("/home", "GET", 200, 42*time.Millisecond)
	m.RecordRequest("/api/users", "POST", 500, 1*time.Second)
}

func TestHandler_NonNil(t *testing.T) {
	m := New()
	h := m.Handler()
	if h == nil {
		t.Fatal("Handler() returned nil")
	}
}

func TestName(t *testing.T) {
	m := New()
	if m.Name() != "prometheus" {
		t.Fatalf("expected 'prometheus', got %q", m.Name())
	}
}
