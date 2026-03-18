// Package react provides a React Server Components (RSC) renderer for Pola.
// It implements streaming SSR via the RSC Flight wire protocol.
//
//go:build react
package react

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/polagonow/pola/core"
)

// Renderer implements the React SSR renderer.
type Renderer struct {
	pool      core.SSRPool
	shell     core.HTMLShell
	streaming bool
}

// New creates a React renderer.
func New() *Renderer { return &Renderer{} }

func (r *Renderer) Name() string             { return "react" }
func (r *Renderer) FileExtensions() []string { return []string{".tsx", ".jsx", ".ts", ".js"} }
func (r *Renderer) Capabilities() []core.Capability {
	return []core.Capability{"streaming", "rsc"}
}

// LoadBundle implements core.BundleLoader. It asks the engine to create an
// SSRPool from the compiled server bundle.
func (r *Renderer) LoadBundle(engine core.JSEngine, bundle []byte) error {
	factory, ok := engine.(core.SSRPoolFactory)
	if !ok {
		return fmt.Errorf("react renderer: engine %q does not implement SSRPoolFactory", engine.Name())
	}
	pool, err := factory.NewSSRPool(bundle)
	if err != nil {
		return fmt.Errorf("react renderer: create SSR pool: %w", err)
	}
	r.pool = pool
	return nil
}

// Render implements core.Renderer. It acquires a VM from the pool, drives the
// RSC Flight render, streams all chunks to the writer embedded in ctx, and
// releases the VM.
func (r *Renderer) Render(ctx context.Context, req core.RenderRequest) (core.RenderResult, error) {
	if r.pool == nil {
		return core.RenderResult{}, fmt.Errorf("react renderer: VM pool not configured")
	}

	vm := r.pool.Acquire()
	defer r.pool.Release(vm)

	// Apply per-request injectors (Go → JS bridge).
	for _, inj := range req.Injectors {
		if err := inj.Inject(ctx, vm); err != nil {
			return core.RenderResult{}, fmt.Errorf("react renderer: inject %s: %w", inj.Name(), err)
		}
	}

	if err := vm.SetRequestContext(req.RequestContext); err != nil {
		return core.RenderResult{}, fmt.Errorf("react renderer: set context: %w", err)
	}

	propsJSON, err := json.Marshal(req.Props)
	if err != nil {
		return core.RenderResult{}, fmt.Errorf("react renderer: marshal props: %w", err)
	}

	_, err = vm.CallRenderFunction(req.Route.Export, string(propsJSON))
	if err != nil {
		return core.RenderResult{}, fmt.Errorf("react renderer: call render: %w", err)
	}

	// Streaming result — the caller must use RenderToWriter for the full
	// streaming path where a concrete StreamWriter is available.
	return core.RenderResult{
		ContentType: ContentType,
		Streaming:   true,
	}, nil
}

// ContentType is the MIME type for the RSC Flight wire format.
const ContentType = "text/x-component"

// IsStreamingRequest reports whether the request carries the RSC Flight
// Content-Type header, meaning the client expects the raw stream.
func IsStreamingRequest(r *http.Request) bool {
	return r.Header.Get("Content-Type") == "text/x-component"
}

// RenderToWriter acquires a VM, performs a full RSC Flight render, and
// streams all chunks to w. This is the primary render path used by the
// Pola pipeline when it has a concrete StreamWriter available.
func (r *Renderer) RenderToWriter(ctx context.Context, req core.RenderRequest, w core.StreamWriter) error {
	if r.pool == nil {
		return fmt.Errorf("react renderer: VM pool not configured")
	}

	vm := r.pool.Acquire()
	defer r.pool.Release(vm)

	// Apply per-request injectors.
	for _, inj := range req.Injectors {
		if err := inj.Inject(ctx, vm); err != nil {
			return fmt.Errorf("react renderer: inject %s: %w", inj.Name(), err)
		}
	}

	if err := vm.SetRequestContext(req.RequestContext); err != nil {
		return fmt.Errorf("react renderer: set context: %w", err)
	}

	propsJSON, err := json.Marshal(req.Props)
	if err != nil {
		return fmt.Errorf("react renderer: marshal props: %w", err)
	}

	handle, err := vm.CallRenderFunction(req.Route.Export, string(propsJSON))
	if err != nil {
		return fmt.Errorf("react renderer: call render: %w", err)
	}

	wroteAny, err := vm.DrainStream(handle, w)
	if err != nil {
		return fmt.Errorf("react renderer: drain stream: %w", err)
	}
	if !wroteAny {
		return fmt.Errorf("react renderer: no Flight output written")
	}
	return nil
}

// WithPool returns a copy of the renderer that uses pool for VM acquisition.
func (r *Renderer) WithPool(pool core.SSRPool) *Renderer {
	c := *r
	c.pool = pool
	return &c
}

// WithShell returns a copy of the renderer that uses shell for HTML rendering.
func (r *Renderer) WithShell(shell core.HTMLShell) *Renderer {
	c := *r
	c.shell = shell
	return &c
}

// GenerateEntry implements the internal entryGenerator interface.
// It converts a DiscoveryResult into the in-memory server-entry TypeScript
// source that the bundler compiles into the server VM bundle.
func (r *Renderer) GenerateEntry(discovery core.DiscoveryResult) (string, error) {
	gen := &EntryGenerator{}
	return gen.Generate(EntryGenConfig{
		Pages:              discovery.Pages,
		AppDir:             discovery.AppDir,
		GlobalErrorPath:    discovery.GlobalError,
		GlobalNotFoundPath: discovery.GlobalNotFound,
	})
}

// BundleConditions returns the esbuild conditions for the React RSC server pass.
func (r *Renderer) BundleConditions() []string {
	return []string{"react-server", "browser", "module", "default"}
}

// ClientEntry returns the package specifier for the React client hydration entry.
func (r *Renderer) ClientEntry() string {
	return "@pola/react/client"
}
