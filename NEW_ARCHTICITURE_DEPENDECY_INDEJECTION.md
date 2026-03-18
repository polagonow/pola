# 🧠 1. Core Idea

> **Runtime Injection Plugin:**
> Provides Go functions, structs, and constants to a VM via **DI container**, before executing any JS code.

* Supports multiple DI frameworks (`dig`, `do`, `sarulabs/di`)
* Works with any JS engine plugin (`Goja`, `V8`)
* Fully modular and pluggable

---

# 🧩 2. Plugin Interface

```go
type RuntimeInjectionPlugin interface {
    Name() string

    // Inject dependencies into a given JS runtime
    Inject(ctx context.Context, runtime JSRuntime) error

    // Optional: declare capabilities (functions, structs, constants)
    Capabilities() []InjectionCapability
}
```

---

## JSRuntime Interface (unchanged)

```go
type JSRuntime interface {
    Eval(script string) (any, error)
    Call(fn string, args ...any) (any, error)
    Set(name string, value any) error
    Dispose()
}
```

---

# 🧩 3. Example: Using Uber Dig

```go
import (
    "github.com/dop251/goja"
    "go.uber.org/dig"
)

type MultiplyService struct{}

func (m *MultiplyService) Multiply(a, b int) int {
    return a * b
}

type DigInjectionPlugin struct {
    Container *dig.Container
}

func (p *DigInjectionPlugin) Inject(ctx context.Context, runtime JSRuntime) error {
    return p.Container.Invoke(func(m *MultiplyService) {
        runtime.Set("multiply", func(a, b int) int {
            return m.Multiply(a, b)
        })
    })
}

func (p *DigInjectionPlugin) Name() string { return "dig-inject" }
```

> Any Go struct/function registered in `dig.Container` can be exposed to JS.

---

# 🧩 4. Optional: Using `samber/do` or `sarulabs/di`

* **samber/do:** Lightweight, fast, no reflection. Same pattern: inject objects → call `runtime.Set()`.
* **sarulabs/di:** Compile-time style DI, good for single-binary deployment.

**Pattern:**

```go
runtime.Set("serviceName", container.Resolve(&Service{}))
```

---

# 🧠 5. How It Fits in Your Engine

### Initialization Flow

```text
Engine Startup
   ↓
Load JS Engine Plugin (Goja/V8)
   ↓
Apply RuntimeInjectionPlugin
   ↓
Set functions/structs/constants
   ↓
Execute JS scripts
```

* Works for SSR renderer plugins, bundlers, CSS processors (if they need JS), or dev tools.
* Hot reload is unaffected: re-inject when VM is reset.

---

# 🧩 6. Example with Goja + MultiplyService

```go
vm := goja.New()

// Inject via RuntimeInjectionPlugin
plugin := &DigInjectionPlugin{Container: container}
plugin.Inject(ctx, &GojaRuntime{vm})

// JS can now call:
result, _ := vm.RunString(`multiply(10, 5)`)
fmt.Println(result.Export()) // Output: 50
```

---

# 🧩 7. Plugin Structure

```text
/plugins
  /runtime_injection
    /dig
      plugin.go
    /do
      plugin.go
    /sarulabs
      plugin.go
```

---

# 🔗 8. Engine + Runtime Injection Integration

* Renderer plugin asks **JS engine plugin** to provide runtime
* Orchestrator calls **all active RuntimeInjectionPlugins** → inject functions/structs
* JS runtime now has access to Go functionality in a **sandboxed and controlled** way

---

# ⚡ 9. Hot Reload Consideration

* On VM reset (hot reload of a page/component):

  1. Dispose old runtime
  2. Create new runtime
  3. Re-run **all RuntimeInjectionPlugins** to re-inject functions/structs
  4. Run JS

* Works for Goja, V8, or any engine with `Set`-style API

---

# 🧭 10. Advantages

1. Full **dependency injection support** for JS VMs
2. Works with multiple DI frameworks (`dig`, `do`, `sarulabs/di`)
3. Enables clean **function + struct exposure** without polluting global scope
4. Fully **plugin-driven** → easy to swap DI library later
5. Supports **single-binary mode**: structs/functions embedded in code

---

# 🏗️ 11. Next Steps

1. Create **RuntimeInjectionPlugin interface** in `engine/core`.
2. Implement **DigInjectionPlugin** (MVP).
3. Optionally implement **doInjectionPlugin** and **sarulabsInjectionPlugin**.
4. Modify orchestrator: after creating VM, **call all active RuntimeInjectionPlugins** before JS execution.
5. Add **tests**: inject functions + structs + constants → call from JS → validate output.
6. Extend hot reload: re-inject after VM reset.

---

✅ This gives you **full runtime injection capabilities**, ready for **SSR, bundling, and hot reload**, fully integrated into your **plugin-driven engine**.