package sobek

import (
	"fmt"

	"github.com/polagonow/pola/core"
)

// Plugin returns the sobek JS engine plugin (grafana/sobek — a maintained goja
// fork, pure Go). The engine compiles the server bundle later via NewSSRPool.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "sobek",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.JSEngine](r, &Engine{})
		},
	}
}

// ── core.SSRRuntime / core.SSRPoolFactory bridge ────────────────────────────────
// Bridges sobek's StartRender + drainSession to the current renderer contract,
// mirroring the goja engine.

type sobekStreamHandle struct{ sess StreamSession }

func (h *sobekStreamHandle) IsNil() bool { return h == nil }

// CallRenderFunction implements core.SSRRuntime.
func (r *Runtime) CallRenderFunction(exportName, propsJSON string) (core.StreamHandle, error) {
	sess, err := r.StartRender(exportName, propsJSON)
	if err != nil {
		return nil, err
	}
	return &sobekStreamHandle{sess: sess}, nil
}

// DrainStream implements core.SSRRuntime.
func (r *Runtime) DrainStream(handle core.StreamHandle, w core.StreamWriter) (bool, error) {
	h, ok := handle.(*sobekStreamHandle)
	if !ok {
		return false, fmt.Errorf("sobek: DrainStream: unexpected handle type %T", handle)
	}
	return r.drainSession(h.sess, w)
}

var _ core.SSRRuntime = (*Runtime)(nil)

// NewSSRPool implements core.SSRPoolFactory.
func (e *Engine) NewSSRPool(bundle []byte) (core.SSRPool, error) {
	pool, err := NewVMPool(string(bundle), e.logger)
	if err != nil {
		return nil, err
	}
	return &sobekSSRPool{pool: pool}, nil
}

type sobekSSRPool struct{ pool *VMPool }

func (p *sobekSSRPool) Acquire() (core.SSRRuntime, error) { return p.pool.Acquire() }
func (p *sobekSSRPool) Release(rt core.SSRRuntime) {
	if r, ok := rt.(*Runtime); ok {
		p.pool.Release(r)
	}
}

var _ core.SSRPoolFactory = (*Engine)(nil)
