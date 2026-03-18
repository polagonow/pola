package pprof

import (
	"testing"
)

func TestHandler_NonNil(t *testing.T) {
	s := New()
	h := s.Handler()
	if h == nil {
		t.Fatal("Handler() returned nil")
	}
}

func TestName(t *testing.T) {
	s := New()
	if s.Name() != "pprof" {
		t.Fatalf("expected 'pprof', got %q", s.Name())
	}
}
