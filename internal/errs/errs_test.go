package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/polagonow/pola/internal/errs"
)

// ---------------------------------------------------------------------------
// Format
// ---------------------------------------------------------------------------

func TestFormat_SingleError(t *testing.T) {
	err := errors.New("something broke")
	got := errs.Format(err)
	want := "something broke"
	if got != want {
		t.Errorf("Format(single) =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormat_ThreeLevelChain(t *testing.T) {
	root := errors.New("file not found: app/page.tsx")
	mid := fmt.Errorf("esbuild: resolve failed: %w", root)
	top := fmt.Errorf("pola: build[bundle]: %w", mid)

	got := errs.Format(top)
	want := "file not found: app/page.tsx\n  esbuild: resolve failed\n    pola: build[bundle]"
	if got != want {
		t.Errorf("Format(3-level) =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormat_BuildError(t *testing.T) {
	inner := errors.New("esbuild crashed")
	be := &buildError{Stage: "bundle", Err: inner}

	got := errs.Format(be)
	want := "esbuild crashed\n  pola: build[bundle]"
	if got != want {
		t.Errorf("Format(BuildError) =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormat_JoinedErrors(t *testing.T) {
	e1 := fmt.Errorf("outer1: %w", errors.New("inner1"))
	e2 := errors.New("standalone")
	joined := errs.Join(e1, e2)

	got := errs.Format(joined)
	want := "inner1\n  outer1\n\nstandalone"
	if got != want {
		t.Errorf("Format(joined) =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormat_Nil(t *testing.T) {
	if got := errs.Format(nil); got != "" {
		t.Errorf("Format(nil) = %q, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// Join
// ---------------------------------------------------------------------------

func TestJoin_AllNils(t *testing.T) {
	if got := errs.Join(nil, nil, nil); got != nil {
		t.Errorf("Join(nil, nil, nil) = %v, want nil", got)
	}
}

func TestJoin_OneNonNil(t *testing.T) {
	orig := errors.New("only")
	got := errs.Join(nil, orig, nil)
	if got != orig {
		t.Errorf("Join(nil, err, nil) did not return the original error")
	}
}

func TestJoin_Multiple(t *testing.T) {
	e1 := errors.New("first")
	e2 := errors.New("second")
	joined := errs.Join(e1, e2)

	if joined == nil {
		t.Fatal("Join(e1, e2) = nil")
	}
	if !errors.Is(joined, e1) {
		t.Error("errors.Is(joined, e1) = false")
	}
	if !errors.Is(joined, e2) {
		t.Error("errors.Is(joined, e2) = false")
	}
	want := "first; second"
	if got := joined.Error(); got != want {
		t.Errorf("joined.Error() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Wrap
// ---------------------------------------------------------------------------

func TestWrap_NilErr(t *testing.T) {
	if got := errs.Wrap(nil, "should not appear"); got != nil {
		t.Errorf("Wrap(nil, ...) = %v, want nil", got)
	}
}

func TestWrap_NonNil(t *testing.T) {
	inner := errors.New("bad thing")
	wrapped := errs.Wrap(inner, "context")
	if wrapped == nil {
		t.Fatal("Wrap returned nil")
	}
	want := "context: bad thing"
	if got := wrapped.Error(); got != want {
		t.Errorf("Wrap().Error() = %q, want %q", got, want)
	}
	if !errors.Is(wrapped, inner) {
		t.Error("errors.Is(Wrap(err), err) = false")
	}
}

// ---------------------------------------------------------------------------
// helpers — local BuildError mirror to avoid importing core
// ---------------------------------------------------------------------------

type buildError struct {
	Stage string
	Err   error
}

func (e *buildError) Error() string {
	return fmt.Sprintf("pola: build[%s]: %v", e.Stage, e.Err)
}
func (e *buildError) Unwrap() error { return e.Err }
