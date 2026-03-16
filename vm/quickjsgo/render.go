package quickjsgo

import (
	"encoding/json"
	"fmt"

	quickjs "github.com/buke/quickjs-go"
	"gojsx/framework"
)

// RenderSession holds the parameters for a single render.
// The actual JS execution happens lazily in DrainStream.
type RenderSession struct {
	ExportName string
	PropsJSON  string
}

// StartRender stores the render parameters. The JS call happens in DrainStream.
func StartRender(_ *VM, exportName, propsJSON string) (*RenderSession, error) {
	return &RenderSession{ExportName: exportName, PropsJSON: propsJSON}, nil
}

// DrainStream installs a per-request __outputChunk__ Go function, then calls
// __renderAsync__(exportName, propsJSON) via ctx.Await. Each time JS produces a
// chunk it calls __outputChunk__ synchronously — because Go-backed functions run
// on the JS thread during ctx.Await, this flushes each chunk to w immediately,
// enabling React Suspense streaming.
func DrainStream(vm *VM, w framework.StreamWriter, sess *RenderSession) (wroteAny bool, err error) {
	var runErr error

	vm.run(func() {
		// Install the per-request output sink. __outputChunk__ is called
		// synchronously from JS during ctx.Await, so w is written incrementally.
		outputFn := vm.ctx.NewFunction(func(qCtx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) > 0 && args[0].IsString() {
				if chunk := args[0].String(); len(chunk) > 0 {
					w.WriteRaw([]byte(chunk)) //nolint:errcheck
					w.Flush()
					wroteAny = true
				}
			}
			return qCtx.NewUndefined()
		})
		g := vm.ctx.Globals()
		g.Set("__outputChunk__", outputFn) // ownership transferred; do NOT free outputFn or g

		// Evaluate the async render — returns a pending Promise.
		exportLit, _ := json.Marshal(sess.ExportName)
		propsLit, _ := json.Marshal(sess.PropsJSON)
		script := fmt.Sprintf("__renderAsync__(%s, %s)", exportLit, propsLit)

		promise := vm.ctx.Eval(script, quickjs.EvalFileName("render.js"))
		defer promise.Free()

		if promise.IsException() {
			runErr = fmt.Errorf("quickjsgo: render eval: %w", vm.ctx.Exception())
			return
		}

		// ctx.Await drives the event loop:
		//   - JS_ExecutePendingJob  — runs native Promise continuations (async components)
		//   - ctx.ProcessJobs()     — runs Go-scheduled bridge-function callbacks
		// During each iteration, __renderAsync__ calls __outputChunk__ synchronously,
		// flushing each chunk to w before the Promise fully resolves.
		result := vm.ctx.Await(promise)
		defer result.Free()

		if result.IsException() {
			runErr = fmt.Errorf("quickjsgo: render await: %w", vm.ctx.Exception())
			return
		}

		// Release the closure's reference to w so the pool VM doesn't retain it.
		clearRet := vm.ctx.Eval("__outputChunk__ = undefined;", quickjs.EvalFileName("clear_output.js"))
		clearRet.Free()
	})

	return wroteAny, runErr
}
