// Package errs provides structured error chain formatting for the Pola framework.
//
// Deeply nested wrapped errors produce flat, hard-to-read strings. This package
// reverses and indents error chains so the root cause appears first.
package errs

import (
	"errors"
	"fmt"
	"strings"
)

// Format walks the error chain and returns a human-readable, indented string
// with the root cause first. For joined errors each independent group is
// separated by a blank line.
func Format(err error) string {
	if err == nil {
		return ""
	}

	// Handle joined errors (multiple independent chains).
	if uw, ok := err.(interface{ Unwrap() []error }); ok {
		children := uw.Unwrap()
		var parts []string
		for _, child := range children {
			parts = append(parts, Format(child))
		}
		return strings.Join(parts, "\n\n")
	}

	// Collect the chain.
	var chain []string
	for e := err; e != nil; e = errors.Unwrap(e) {
		chain = append(chain, message(e))
	}

	// Reverse so root cause is first.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	var b strings.Builder
	for i, msg := range chain {
		if i > 0 {
			b.WriteByte('\n')
			b.WriteString(strings.Repeat("  ", i))
		}
		b.WriteString(msg)
	}
	return b.String()
}

// message returns the error's own message without any wrapped suffix.
// It strips the wrapped error text that appears after the last ": " separator.
func message(err error) string {
	inner := errors.Unwrap(err)
	if inner == nil {
		return err.Error()
	}
	full := err.Error()
	innerMsg := inner.Error()
	// Trim the wrapped portion. The standard library uses ": " as the
	// separator between wrapper and wrapped messages.
	if idx := strings.LastIndex(full, ": "+innerMsg); idx >= 0 {
		return full[:idx]
	}
	return full
}

// joinedError holds multiple independent errors.
type joinedError struct {
	errs []error
}

func (e *joinedError) Error() string {
	msgs := make([]string, len(e.errs))
	for i, err := range e.errs {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

func (e *joinedError) Unwrap() []error {
	return e.errs
}

// Join returns an error that wraps the given errors. Nil errors are filtered
// out. If no non-nil errors remain, nil is returned. If exactly one non-nil
// error remains, it is returned unwrapped.
func Join(errs ...error) error {
	var nonNil []error
	for _, err := range errs {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}
	switch len(nonNil) {
	case 0:
		return nil
	case 1:
		return nonNil[0]
	default:
		return &joinedError{errs: nonNil}
	}
}

// Wrap returns fmt.Errorf("%s: %w", msg, err). If err is nil, Wrap returns nil.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
