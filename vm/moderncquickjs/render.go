package moderncquickjs

import (
	"encoding/json"
	"fmt"
	"time"

	"gojsx/framework"

	mquickjs "modernc.org/quickjs"
)

// RenderSession holds parameters for a single render.
// Actual JS execution happens in DrainStream.
type RenderSession struct {
	ExportName string
	PropsJSON  string
}

// startRender stores the render parameters.
func startRender(_ *VM, exportName, propsJSON string) (*RenderSession, error) {
	return &RenderSession{ExportName: exportName, PropsJSON: propsJSON}, nil
}

// DrainStream drives the React Flight render loop from Go, matching the
// goja/sobek pattern: each vm.run is a separate event-loop task so the
// checkpoint (drainNativeJobs + __drainMicrotasks__) fires between iterations,
// allowing async server components to complete.
func DrainStream(vm *VM, w framework.StreamWriter, sess *RenderSession) (wroteAny bool, err error) {
	// Start render: store stream + TextDecoder in JS globals.
	vm.run(func() {
		vm.currentWriter = w
		vm.wroteAny = false
		exportLit, _ := json.Marshal(sess.ExportName)
		propsLit, _ := json.Marshal(sess.PropsJSON)
		if _, e := vm.inner.Eval(
			fmt.Sprintf("__startRender__(%s, %s)", exportLit, propsLit),
			mquickjs.EvalGlobal,
		); e != nil {
			err = fmt.Errorf("moderncquickjs: start render: %w", e)
		}
	})
	if err != nil {
		return false, err
	}

	// Pull loop: each vm.run is a separate event-loop task.
	// The checkpoint fires between iterations, draining native JS jobs and
	// custom microtasks so async server components can advance.
	for {
		var done bool
		var iterWrote bool
		vm.run(func() {
			vm.wroteAny = false
			result, e := vm.inner.Eval("__pullOnce__()", mquickjs.EvalGlobal)
			if e != nil {
				err = fmt.Errorf("moderncquickjs: render: %w", e)
				return
			}
			done, _ = result.(bool)
			iterWrote = vm.wroteAny
		})
		wroteAny = wroteAny || iterWrote
		if err != nil {
			break
		}
		if done {
			break
		}
		if !iterWrote {
			time.Sleep(5 * time.Millisecond)
		}
	}

	// Cleanup: release the writer reference and clear JS stream globals.
	vm.run(func() {
		vm.currentWriter = nil
		vm.wroteAny = false
		evalOrErr(vm.inner, "__clearRenderStream__()") //nolint:errcheck
	})

	return wroteAny, err
}
