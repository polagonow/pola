package react

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/mailer"
)

// Renderer renders react.email components via a pool of JS runtimes.
// Each runtime has the email bundle loaded and exposes a __renderEmail__ global.
type Renderer struct {
	engine core.JSEngine
	logger core.Logger
	bundle []byte

	mu   sync.Mutex
	pool []core.JSRuntime
}

// Plugin returns a core.Plugin that registers the react.email renderer
// as the mailer.EmailRenderer in the DI container.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "mailer.renderer.react",
		Fn: func(r *core.Registry) {
			engine := core.MustInvoke[core.JSEngine](r)
			logger, _ := core.Invoke[core.Logger](r)
			renderer := &Renderer{engine: engine, logger: logger}
			core.ProvideValue[mailer.EmailRenderer](r, renderer)
			core.ProvideValue[*Renderer](r, renderer)
		},
	}
}

// LoadBundle stores the compiled email bundle. Call this after bundling
// the app/mailers/ directory. Subsequent RenderEmail calls will create
// runtimes from this bundle.
func (r *Renderer) LoadBundle(bundle []byte) error {
	if len(bundle) == 0 {
		return fmt.Errorf("mailer: empty email bundle")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// Discard any existing runtimes — the bundle changed.
	for _, rt := range r.pool {
		rt.Dispose()
	}
	r.pool = nil
	r.bundle = bundle
	return nil
}

// RenderEmail renders a react.email template to HTML and plain text.
func (r *Renderer) RenderEmail(ctx context.Context, templateName, layoutName string, props map[string]any) (string, string, error) {
	r.mu.Lock()
	bundle := r.bundle
	r.mu.Unlock()

	if len(bundle) == 0 {
		return "", "", fmt.Errorf("mailer: no email bundle loaded")
	}

	rt, err := r.acquire(ctx)
	if err != nil {
		return "", "", fmt.Errorf("mailer: acquire runtime: %w", err)
	}
	defer r.release(rt)

	propsJSON, err := json.Marshal(props)
	if err != nil {
		return "", "", fmt.Errorf("mailer: marshal props: %w", err)
	}

	result, err := rt.Call("__renderEmail__", templateName, layoutName, string(propsJSON))
	if err != nil {
		return "", "", fmt.Errorf("mailer: render %s: %w", templateName, err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("mailer: unexpected render result type %T", result)
	}

	html, _ := m["html"].(string)
	text, _ := m["text"].(string)
	return html, text, nil
}

func (r *Renderer) acquire(ctx context.Context) (core.JSRuntime, error) {
	r.mu.Lock()
	if n := len(r.pool); n > 0 {
		rt := r.pool[n-1]
		r.pool = r.pool[:n-1]
		r.mu.Unlock()
		return rt, nil
	}
	bundle := r.bundle
	r.mu.Unlock()

	rt, err := r.engine.NewRuntime(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := rt.Eval(string(bundle)); err != nil {
		rt.Dispose()
		return nil, fmt.Errorf("mailer: eval email bundle: %w", err)
	}
	return rt, nil
}

func (r *Renderer) release(rt core.JSRuntime) {
	r.mu.Lock()
	r.pool = append(r.pool, rt)
	r.mu.Unlock()
}
