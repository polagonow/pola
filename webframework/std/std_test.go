package std

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/polagonow/pola/core"
)

func TestHandlerErrorStatusMapping(t *testing.T) {
	f := New()
	f.Handle(http.MethodGet, "/missing", func(core.Context) error {
		return core.WithStatus(http.StatusNotFound, errors.New("no such thing"))
	})
	f.Handle(http.MethodGet, "/boom", func(core.Context) error {
		return errors.New("kaboom")
	})
	f.Handle(http.MethodGet, "/ok", func(c core.Context) error {
		return c.JSON(http.StatusOK, core.M{"ok": true})
	})
	h := f.Handler()

	do := func(path string) int {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Code
	}

	if got := do("/missing"); got != http.StatusNotFound {
		t.Errorf("mapped error status = %d, want 404", got)
	}
	if got := do("/boom"); got != http.StatusInternalServerError {
		t.Errorf("plain error status = %d, want 500", got)
	}
	if got := do("/ok"); got != http.StatusOK {
		t.Errorf("ok status = %d, want 200", got)
	}
}

func TestHandlerWroteResponseNotOverridden(t *testing.T) {
	f := New()
	f.Handle(http.MethodGet, "/wrote", func(c core.Context) error {
		_ = c.JSON(http.StatusAccepted, core.M{"queued": true})
		return errors.New("late error") // must not override the already-sent 202
	})
	h := f.Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wrote", nil))
	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202 (handler already wrote the response)", rec.Code)
	}
}
