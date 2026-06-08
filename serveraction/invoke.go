package serveraction

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
)

// Runner is the runtime capability needed to invoke a server action: calling a
// global function and awaiting a returned Promise while driving the event loop.
// Implemented by engine/goja's Runtime (CallAwait).
type Runner interface {
	CallAwait(fn string, args ...any) (any, error)
}

// Invoke executes a server action in a VM acquired from pool. It applies the
// per-request injectors and request context, calls the bundle's
// __invokeServerAction__ helper, awaits its Promise, and decodes the resulting
// JSON envelope. The VM is always released back to the pool.
//
// This is renderer-agnostic: both nativersc and react acquire goja runtimes from
// a core.SSRPool, so their ServerActionInvoker implementations delegate here.
func Invoke(ctx context.Context, pool core.SSRPool, in core.InvokeInput) (core.InvokeOutput, error) {
	if pool == nil {
		return core.InvokeOutput{}, fmt.Errorf("serveraction: VM pool not configured")
	}
	vm, err := pool.Acquire()
	if err != nil {
		return core.InvokeOutput{}, fmt.Errorf("serveraction: acquire VM: %w", err)
	}
	defer pool.Release(vm)

	for _, inj := range in.Injectors {
		if err := inj.Inject(ctx, vm); err != nil {
			return core.InvokeOutput{}, fmt.Errorf("serveraction: inject %s: %w", inj.Name(), err)
		}
	}
	if err := vm.SetRequestContext(in.RequestContext); err != nil {
		return core.InvokeOutput{}, fmt.Errorf("serveraction: set context: %w", err)
	}

	runner, ok := vm.(Runner)
	if !ok {
		return core.InvokeOutput{}, fmt.Errorf("serveraction: runtime %T cannot await actions (use the goja engine)", vm)
	}

	argsJSON, err := json.Marshal(in.Args)
	if err != nil {
		return core.InvokeOutput{}, fmt.Errorf("serveraction: marshal args: %w", err)
	}

	res, err := runner.CallAwait(globals.InvokeServerActionFn, in.ID, in.ExportName, string(argsJSON))
	if err != nil {
		return core.InvokeOutput{}, err
	}
	envStr, ok := res.(string)
	if !ok {
		return core.InvokeOutput{}, fmt.Errorf("serveraction: action returned %T, want JSON string", res)
	}

	var env struct {
		Success  bool            `json:"success"`
		Result   json.RawMessage `json:"result"`
		Error    string          `json:"error"`
		Redirect string          `json:"redirect"`
	}
	if err := json.Unmarshal([]byte(envStr), &env); err != nil {
		return core.InvokeOutput{}, fmt.Errorf("serveraction: decode action envelope: %w", err)
	}
	return core.InvokeOutput{
		Success:  env.Success,
		Result:   env.Result,
		Error:    env.Error,
		Redirect: env.Redirect,
	}, nil
}
