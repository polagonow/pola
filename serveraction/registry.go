package serveraction

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Module is a discovered 'use server' module: its registry-key prefix and the
// absolute path the server entry imports.
type Module struct {
	// ModuleID is the path under the app dir without extension (registry key
	// prefix). The full action key is ModuleID + ":" + exportName.
	ModuleID string
	// AbsPath is the absolute path to the module file.
	AbsPath string
}

// Scan walks appDir for .ts/.tsx/.js/.jsx files carrying a top-level
// 'use server' directive and returns them as registry modules. node_modules,
// dotfiles, and the public dir are skipped. The returned IDs match the bundler's
// (both use ModuleID), so the server registry and the client stubs agree. This
// mirrors rari's scan_for_server_actions.
func Scan(appDir string) []Module {
	if appDir == "" {
		return nil
	}
	absAppDir, err := filepath.Abs(appDir)
	if err != nil {
		return nil
	}
	var mods []Module
	_ = filepath.WalkDir(absAppDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if path != absAppDir && (name == "node_modules" || name == "public" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".d.ts") {
			return nil
		}
		switch filepath.Ext(path) {
		case ".ts", ".tsx", ".js", ".jsx":
		default:
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil || !HasUseServerDirective(data) {
			return nil
		}
		mods = append(mods, Module{ModuleID: ModuleID(absAppDir, path), AbsPath: path})
		return nil
	})
	return mods
}

// ImportLines returns `import * as __sa_N__ from "./rel"` lines (relative to
// absAppDir) for each module, in order. They must be emitted at module top so
// RegistryJS (which references __sa_N__ by the same index) can register them.
func ImportLines(modules []Module, absAppDir string) string {
	var b strings.Builder
	for i, m := range modules {
		abs, _ := filepath.Abs(m.AbsPath)
		rel, err := filepath.Rel(absAppDir, abs)
		if err != nil {
			rel = abs
		}
		fmt.Fprintf(&b, "import * as __sa_%d__ from %q;\n", i, "./"+filepath.ToSlash(rel))
	}
	return b.String()
}

// RegistryJS returns the JS that installs the server-action registry, registers
// each module's exports, and defines the lookup/invoke helpers. It must be
// emitted after ImportLines. The registration hooks into
// react-server-dom-webpack via __pola_registerServerReference__ when that global
// is defined (the react renderer imports it); under nativersc it is absent and
// the call is skipped.
func RegistryJS(modules []Module) string {
	var b strings.Builder
	b.WriteString(registryPreludeJS)
	for i, m := range modules {
		fmt.Fprintf(&b, "__pola_register_actions__(%q, __sa_%d__);\n", m.ModuleID, i)
	}
	b.WriteString(registryHelpersJS)
	return b.String()
}

// StubJS returns the client-side stub source for a 'use server' module: each
// export becomes a createServerReference that POSTs to /_pola/action. Shared by
// every renderer's bundle plugin provider.
func StubJS(moduleID string, exports []string) string {
	var b strings.Builder
	b.WriteString(`import { createServerReference } from "@pola/actions/server-reference";` + "\n")
	for _, name := range exports {
		key := moduleID + ":" + name
		if name == "default" {
			fmt.Fprintf(&b, "const __pola_default = createServerReference(%q, %q, %q);\nexport default __pola_default;\n", key, moduleID, name)
		} else {
			fmt.Fprintf(&b, "export const %s = createServerReference(%q, %q, %q);\n", name, key, moduleID, name)
		}
	}
	return b.String()
}

// registryPreludeJS installs __POLA_SERVER_ACTIONS__ and the
// __pola_register_actions__ helper. Each registered function is tagged with the
// react.server.reference markers (so the Go-driven nativersc Flight serializer
// can emit it as a server reference) and, when the react renderer is in use,
// registered with react-server-dom-webpack so its serializer recognizes it too.
// Plain JS (no TS) so it is valid as esbuild TSX input and runnable in goja unit
// tests.
const registryPreludeJS = `
globalThis.__POLA_SERVER_ACTIONS__ = globalThis.__POLA_SERVER_ACTIONS__ || Object.create(null);
function __pola_register_actions__(moduleId, mod) {
  if (!mod) return;
  for (var name in mod) {
    var fn = mod[name];
    if (typeof fn !== "function") continue;
    var key = moduleId + ":" + name;
    try { fn.$$typeof = Symbol.for("react.server.reference"); fn.$$id = key; } catch (e) {}
    try {
      if (typeof __pola_registerServerReference__ === "function") {
        __pola_registerServerReference__(fn, key, null);
      }
    } catch (e) {}
    globalThis.__POLA_SERVER_ACTIONS__[key] = fn;
  }
}
`

// registryHelpersJS installs __getServerFunction__ (rari-style lookup: exact
// key, then ambiguity-checked suffix match) and __invokeServerAction__, the
// Go-callable entry point resolving to a JSON envelope string
// { success, result, error, redirect }.
const registryHelpersJS = `
globalThis.__getServerFunction__ = function (id, exportName) {
  var reg = globalThis.__POLA_SERVER_ACTIONS__;
  var key = id + ":" + exportName;
  if (typeof reg[key] === "function") return reg[key];
  var suffix = ":" + exportName;
  var found = null;
  for (var k in reg) {
    if (k.length >= suffix.length && k.slice(k.length - suffix.length) === suffix) {
      if (found !== null) {
        throw new Error("Ambiguous server function '" + exportName + "': matches '" + found + "' and '" + k + "'. Use the full id.");
      }
      found = k;
    }
  }
  return found !== null ? reg[found] : null;
};

globalThis.__invokeServerAction__ = function (id, exportName, argsJSON) {
  return Promise.resolve().then(function () {
    var fn = globalThis.__getServerFunction__(id, exportName);
    if (typeof fn !== "function") {
      throw new Error("server action not found: " + id + ":" + exportName);
    }
    var args = argsJSON ? JSON.parse(argsJSON) : [];
    return fn.apply(null, args);
  }).then(function (result) {
    if (result && typeof result === "object" && typeof result.redirect === "string") {
      return JSON.stringify({ success: true, result: null, error: null, redirect: result.redirect });
    }
    return JSON.stringify({ success: true, result: result === undefined ? null : result, error: null, redirect: null });
  }, function (err) {
    if (err && err.__pola_redirect__) {
      return JSON.stringify({ success: true, result: null, error: null, redirect: String(err.__pola_redirect__) });
    }
    var msg = (err && err.message) ? err.message : String(err);
    return JSON.stringify({ success: false, result: null, error: msg, redirect: null });
  });
};
`
