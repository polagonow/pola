package core

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestStatusOf(t *testing.T) {
	if got := StatusOf(errors.New("plain")); got != http.StatusInternalServerError {
		t.Errorf("plain err → %d, want 500", got)
	}
	if got := StatusOf(WithStatus(http.StatusNotFound, errors.New("nope"))); got != http.StatusNotFound {
		t.Errorf("WithStatus 404 → %d, want 404", got)
	}
	if got := StatusOf(nil); got != http.StatusInternalServerError {
		t.Errorf("nil → %d, want 500", got)
	}
}

func TestWithStatusUnwraps(t *testing.T) {
	sentinel := errors.New("boom")
	err := WithStatus(http.StatusBadGateway, fmt.Errorf("upstream: %w", sentinel))

	if !errors.Is(err, sentinel) {
		t.Error("WithStatus should preserve the wrapped error for errors.Is")
	}
	if StatusOf(err) != http.StatusBadGateway {
		t.Errorf("StatusOf through a wrap = %d, want 502", StatusOf(err))
	}
	if err.Error() != "upstream: boom" {
		t.Errorf("Error() = %q, want 'upstream: boom'", err.Error())
	}
}

func TestStatusErrorNilErr(t *testing.T) {
	err := WithStatus(http.StatusTeapot, nil)
	if err.Error() != http.StatusText(http.StatusTeapot) {
		t.Errorf("Error() with nil wrapped = %q, want status text", err.Error())
	}
}
