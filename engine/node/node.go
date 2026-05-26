// Package node provides a JSEngine implementation that runs JavaScript via the
// system Node.js binary (exec-based). Use this when you want full Node.js
// compatibility and don't need a single-binary deployment.
//
// Build tag: node
package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/polagonow/pola/core"
)

// Engine runs JS by spawning `node` subprocesses.
type Engine struct {
	Bin string // path to node binary, default "node"
}

// New creates a node Engine. bin is the path to the node binary ("node" or absolute path).
func New(bin string) *Engine {
	if bin == "" {
		bin = "node"
	}
	return &Engine{Bin: bin}
}

// Name implements core.JSEngine.
func (e *Engine) Name() string { return "node" }

// RequiredPolyfills returns nil — Node.js has all web APIs built in.
func (e *Engine) RequiredPolyfills() []core.PolyfillID { return nil }

// NewRuntime implements core.JSEngine.
func (e *Engine) NewRuntime(_ context.Context) (core.JSRuntime, error) {
	return &Runtime{bin: e.Bin, globals: map[string]any{}}, nil
}

// ── Runtime ───────────────────────────────────────────────────────────────────

// Runtime executes JavaScript by spawning node subprocesses.
type Runtime struct {
	bin    string
	bundle string
	// globals holds values registered via Set to be injected before scripts run.
	globals map[string]any
}

// Eval implements core.JSRuntime.
// It runs the script via `node --eval` and returns stdout as a string.
func (r *Runtime) Eval(script string) (any, error) {
	preamble, err := r.buildPreamble()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(r.bin, "--eval", preamble+"\n"+script) //nolint:gosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("node eval: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

// Call implements core.JSRuntime.
// It serialises args to JSON, injects them and the bundle, then calls fn.
func (r *Runtime) Call(fn string, args ...any) (any, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("node call: marshal args: %w", err)
	}
	preamble, err := r.buildPreamble()
	if err != nil {
		return nil, err
	}
	script := fmt.Sprintf(`%s
var __args = %s;
var __result = %s.apply(null, __args);
process.stdout.write(JSON.stringify(__result));`, preamble, string(argsJSON), fn)

	cmd := exec.Command(r.bin, "--eval", script) //nolint:gosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("node call %s: %w\nstderr: %s", fn, err, stderr.String())
	}

	var result any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		// Return raw stdout if not valid JSON.
		return stdout.String(), nil
	}
	return result, nil
}

// Set implements core.JSRuntime.
// Values are stored and injected as globals when Eval/Call runs.
func (r *Runtime) Set(name string, value any) error {
	r.globals[name] = value
	return nil
}

// Dispose implements core.JSRuntime. No-op for the exec-based runtime.
func (r *Runtime) Dispose() {}

// buildPreamble builds a JavaScript snippet that injects globals and the bundle.
func (r *Runtime) buildPreamble() (string, error) {
	var buf bytes.Buffer
	for name, val := range r.globals {
		data, err := json.Marshal(val)
		if err != nil {
			return "", fmt.Errorf("node preamble: marshal global %s: %w", name, err)
		}
		fmt.Fprintf(&buf, "var %s = %s;\n", name, string(data))
	}
	if r.bundle != "" {
		buf.WriteString(r.bundle)
		buf.WriteString("\n")
	}
	return buf.String(), nil
}
