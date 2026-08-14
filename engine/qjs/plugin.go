package qjs

import (
	"fmt"

	"github.com/polagonow/pola/core"
)

// Plugin returns the qjs JS engine plugin (fastschema/qjs — a QuickJS binding).
// The engine compiles the server bundle later via NewSSRPool.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "qjs",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.JSEngine](r, &Engine{})
		},
	}
}

// ── core.SSRRuntime / core.SSRPoolFactory bridge ────────────────────────────────
// Bridges qjs's StartRender + drainSession to the current renderer contract,
// mirroring the goja engine.

type qjsStreamHandle struct{ sess RenderSession }

func (h *qjsStreamHandle) IsNil() bool { return h == nil }

// CallRenderFunction implements core.SSRRuntime.
func (r *Runtime) CallRenderFunction(exportName, propsJSON string) (core.StreamHandle, error) {
	sess, err := r.StartRender(exportName, propsJSON)
	if err != nil {
		return nil, err
	}
	return &qjsStreamHandle{sess: sess}, nil
}

// DrainStream implements core.SSRRuntime.
func (r *Runtime) DrainStream(handle core.StreamHandle, w core.StreamWriter) (bool, error) {
	h, ok := handle.(*qjsStreamHandle)
	if !ok {
		return false, fmt.Errorf("qjs: DrainStream: unexpected handle type %T", handle)
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
	return &qjsSSRPool{pool: pool}, nil
}

type qjsSSRPool struct{ pool *VMPool }

func (p *qjsSSRPool) Acquire() (core.SSRRuntime, error) { return p.pool.Acquire() }
func (p *qjsSSRPool) Release(rt core.SSRRuntime) {
	if r, ok := rt.(*Runtime); ok {
		p.pool.Release(r)
	}
}

var _ core.SSRPoolFactory = (*Engine)(nil)
