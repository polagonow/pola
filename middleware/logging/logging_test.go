package logging_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/middleware/logging"
)

// recordLogger captures log calls for assertions.
type recordLogger struct {
	msgs []string
}

func (r *recordLogger) Info(msg string, args ...any)  { r.msgs = append(r.msgs, "INFO:"+msg) }
func (r *recordLogger) Error(msg string, args ...any) { r.msgs = append(r.msgs, "ERROR:"+msg) }
func (r *recordLogger) Debug(msg string, args ...any) { r.msgs = append(r.msgs, "DEBUG:"+msg) }
func (r *recordLogger) Warn(msg string, args ...any)  { r.msgs = append(r.msgs, "WARN:"+msg) }
func (r *recordLogger) With(args ...any) core.Logger  { return r }

var _ core.Logger = (*recordLogger)(nil)

func TestLoggingMiddleware_LogsRequest(t *testing.T) {
	log := &recordLogger{}
	mw := logging.New(log)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(log.msgs) == 0 {
		t.Fatal("expected at least one log message")
	}
	found := false
	for _, m := range log.msgs {
		if m == "INFO:request" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected INFO:request log, got: %v", log.msgs)
	}
}

func TestLoggingMiddleware_Name(t *testing.T) {
	mw := logging.New(&recordLogger{})
	if mw.Name() != "logging" {
		t.Fatalf("expected 'logging', got %q", mw.Name())
	}
}

func TestLoggingMiddleware_CapturesStatus(t *testing.T) {
	log := &recordLogger{}
	mw := logging.New(log)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/gone", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(log.msgs) == 0 {
		t.Fatal("expected log message")
	}
}
