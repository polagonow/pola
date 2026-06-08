//go:build esbuild

package esbuild

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/polagonow/pola/core"
)

// testServerActionStub is a self-contained client stub generator (no external
// @pola dependency) so the test bundles without node_modules.
func testServerActionStub(_, moduleID string, exports []string) string {
	var b strings.Builder
	b.WriteString("function createServerReference(id, mod, name){ return async function(){ return null; }; }\n")
	for _, name := range exports {
		fmt.Fprintf(&b, "export const %s = createServerReference(%q, %q, %q);\n",
			name, moduleID+":"+name, moduleID, name)
	}
	return b.String()
}

// TestBuild_ServerActions verifies that 'use server' modules are discovered,
// their exports extracted, registered in BundleOutput.ServerActions, and that
// the client bundle ships createServerReference stubs instead of server code.
func TestBuild_ServerActions(t *testing.T) {
	// Resolve symlinks so appDir matches esbuild's symlink-resolved paths
	// (macOS /tmp is a symlink to /private/tmp).
	appDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(appDir, "public", "assets")
	if err := os.MkdirAll(filepath.Join(appDir, "actions"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 'use server' module: two functions + a type-only export (must be excluded).
	action := `"use server"
export async function addTodo(text: string) {
  const SECRET = "__server_only_secret__";
  return text + SECRET;
}
export async function removeTodo(id: string) { return id; }
export interface Todo { id: string }
`
	if err := os.WriteFile(filepath.Join(appDir, "actions", "todos.ts"), []byte(action), 0o644); err != nil {
		t.Fatal(err)
	}

	// Client entry imports an action so the client bundle exercises the stub.
	clientEntry := filepath.Join(appDir, "client-entry.tsx")
	if err := os.WriteFile(clientEntry, []byte(`import { addTodo } from "./actions/todos";
console.log(addTodo);
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Server entry imports the action module so the probe discovers it.
	serverEntry := `import * as todos from "./actions/todos";
(globalThis as any).__todos = todos;
`

	noopProbe := api.Plugin{Name: "noop", Setup: func(api.PluginBuild) {}}

	b := New()
	out, err := b.Build(context.Background(), core.BundleInput{
		AppDir:                 appDir,
		OutDir:                 outDir,
		AssetsURLPath:          "/public/assets",
		ClientEntry:            clientEntry,
		ServerEntryContent:     serverEntry,
		ServerBundleConditions: []string{"browser", "module", "default"},
		ProbePlugins:           []any{noopProbe},
		ServerActionStub:       testServerActionStub,
		Dev:                    true, // no minification → readable stub assertions
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Manifest of valid action ids/exports.
	exports, ok := out.ServerActions["actions/todos"]
	if !ok {
		t.Fatalf("ServerActions missing 'actions/todos'; got %v", out.ServerActions)
	}
	if !contains(exports, "addTodo") || !contains(exports, "removeTodo") {
		t.Errorf("exports = %v, want addTodo + removeTodo", exports)
	}
	if contains(exports, "Todo") {
		t.Errorf("type-only export Todo should not be registered: %v", exports)
	}

	// The server secret must never appear in any client output; a server
	// reference stub must.
	var sawStub bool
	for name, data := range out.ClientFiles {
		if strings.Contains(string(data), "__server_only_secret__") {
			t.Errorf("client file %s leaked server code", name)
		}
		if strings.Contains(string(data), "createServerReference") {
			sawStub = true
		}
	}
	if !sawStub {
		t.Error("expected a createServerReference stub in the client bundle")
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
