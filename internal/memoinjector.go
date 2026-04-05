package internal

import (
	"context"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/engine/polyfill"
)

// MemoInjector wraps a RuntimeInjector with per-request function call
// memoization. When a Go bridge function is called with the same arguments
// during a single render pass, the cached result is returned instead of
// re-executing the function. This prevents redundant Go function calls when
// layouts and pages need the same data (similar to Next.js fetch memoization).
type MemoInjector struct {
	inner core.RuntimeInjector
}

// NewMemoInjector wraps an existing injector with per-request memoization.
func NewMemoInjector(inner core.RuntimeInjector) *MemoInjector {
	return &MemoInjector{inner: inner}
}

func (m *MemoInjector) Name() string                             { return m.inner.Name() }
func (m *MemoInjector) Capabilities() []core.InjectionCapability { return m.inner.Capabilities() }

func (m *MemoInjector) Inject(ctx context.Context, runtime core.JSRuntime) error {
	// First, let the inner injector set up the bridge as usual.
	if err := m.inner.Inject(ctx, runtime); err != nil {
		return err
	}

	// Install memoization wrapper via JS eval. The source lives in
	// engine/polyfill alongside all other framework JS polyfills.
	_, err := runtime.Eval(polyfill.BridgeMemoSrc)
	return err
}

// WrapInjectorsWithMemo wraps a slice of RuntimeInjectors with memoization.
func WrapInjectorsWithMemo(injectors []core.RuntimeInjector) []core.RuntimeInjector {
	wrapped := make([]core.RuntimeInjector, len(injectors))
	for i, inj := range injectors {
		wrapped[i] = NewMemoInjector(inj)
	}
	return wrapped
}
