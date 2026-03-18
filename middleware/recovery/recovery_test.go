package recovery_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/middleware/recovery"
)

type recordLogger struct {
	errors []string
}

func (r *recordLogger) Info(msg string, args ...any)  {}
func (r *recordLogger) Debug(msg string, args ...any) {}
func (r *recordLogger) Warn(msg string, args ...any)  {}
func (r *recordLogger) Error(msg string, args ...any) { r.errors = append(r.errors, msg) }
func (r *recordLogger) With(args ...any) core.Logger  { return r }

var _ core.Logger = (*recordLogger)(nil)

func TestRecovery_CatchesPanic(t *testing.T) {
	log := &recordLogger{}
	mw := recovery.New(log)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// Should not panic.
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if len(log.errors) == 0 {
		t.Fatal("expected error log entry")
	}
}

func TestRecovery_NoPanic_PassesThrough(t *testing.T) {
	log := &recordLogger{}
	mw := recovery.New(log)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if len(log.errors) != 0 {
		t.Fatal("expected no error log entries")
	}
}

func TestRecovery_Name(t *testing.T) {
	mw := recovery.New(&recordLogger{})
	if mw.Name() != "recovery" {
		t.Fatalf("expected 'recovery', got %q", mw.Name())
	}
}
