# 🧠 1. Core Principle

> 👉 File system = **plugin layer**, fully pluggable.
> Renderers, bundlers, CSS plugins, and static assets must **always access files via this FS plugin interface**, never directly from the OS.

* User can pick:

  1. **EmbedFS** (using Go `embed`) → single binary
  2. **OSFS** (read from disk) → external static files

* Optional: support **hybrid mode** (fallback to disk if embed missing).

---

# 🧩 2. FS Plugin Interface

```go
type FSPlugin interface {
    Name() string

    ReadFile(path string) ([]byte, error)
    ReadDir(path string) ([]FSFileInfo, error)
    Exists(path string) bool

    // Optional: watch for hot reload
    Watch(path string, onChange func(string)) error
}
```

---

## FS File Info abstraction

```go
type FSFileInfo struct {
    Name     string
    IsDir    bool
    Size     int64
    ModTime  time.Time
}
```

---

# 🧠 3. FS Plugin Implementations

---

## 3.1 EmbedFS Plugin (single binary)

```go
type EmbedFSPlugin struct {
    fs embed.FS
}

func (e *EmbedFSPlugin) ReadFile(path string) ([]byte, error) {
    return e.fs.ReadFile(path)
}

func (e *EmbedFSPlugin) ReadDir(path string) ([]FSFileInfo, error) {
    entries, _ := e.fs.ReadDir(path)
    var files []FSFileInfo
    for _, entry := range entries {
        files = append(files, FSFileInfo{
            Name: entry.Name(),
            IsDir: entry.IsDir(),
        })
    }
    return files, nil
}

func (e *EmbedFSPlugin) Exists(path string) bool {
    _, err := e.fs.Open(path)
    return err == nil
}
```

> Note: `embed.FS` is read-only; hot reload may be **disabled** or simulated with dev-mode rebuild.

---

## 3.2 OSFS Plugin (external files)

```go
type OSFSPlugin struct {
    Root string
}

func (o *OSFSPlugin) ReadFile(path string) ([]byte, error) {
    return os.ReadFile(filepath.Join(o.Root, path))
}

func (o *OSFSPlugin) ReadDir(path string) ([]FSFileInfo, error) {
    entries, err := os.ReadDir(filepath.Join(o.Root, path))
    if err != nil { return nil, err }
    var files []FSFileInfo
    for _, entry := range entries {
        info, _ := entry.Info()
        files = append(files, FSFileInfo{
            Name: entry.Name(),
            IsDir: entry.IsDir(),
            Size: info.Size(),
            ModTime: info.ModTime(),
        })
    }
    return files, nil
}

func (o *OSFSPlugin) Exists(path string) bool {
    _, err := os.Stat(filepath.Join(o.Root, path))
    return err == nil
}
```

> This supports **hot reload**, because the file system can be watched with `fsnotify`.

---

# 🧩 4. FS Plugin Integration

### Plugin Registry

```go
type Registry struct {
    FS FSPlugin
    // other plugins...
}
```

* Renderer → reads templates via `FSPlugin.ReadFile()`
* CSS plugin → reads Tailwind config or Sass files via FS
* Bundler → reads entry points via FS
* Router → scans `/app` directory via FS
* HotReloadManager → subscribes to `FSPlugin.Watch()` events

---

# 🧠 5. Hot Reload Considerations

* **EmbedFS**: read-only → hot reload can only work by rebuilding binary
* **OSFS**: full watch → supports incremental rebuild + live reload

> Rule: Dev mode chooses OSFS for hot reload; Prod mode can embed.

---

# 🧩 6. FS Configuration

```go
type FSConfig struct {
    Type string // "embed" or "os"
    Root string // only for OSFS
}
```

**Engine initialization example:**

```go
var fsPlugin FSPlugin
if cfg.Type == "embed" {
    fsPlugin = &EmbedFSPlugin{fs: embeddedAssets}
} else {
    fsPlugin = &OSFSPlugin{Root: cfg.Root}
}
registry.FS = fsPlugin
```

---

# 🧭 7. Pipeline Flow (with FS)

```text
FS Plugin (embed or OS)
      ↓
Router scans /app → RouteMatch
      ↓
Bundler reads entry points
      ↓
CSS plugin reads configs
      ↓
Renderer reads templates/components
      ↓
Cache stores pages/fragments
      ↓
HotReloadManager listens (OSFS only)
```

* All file access goes **through FSPlugin abstraction**.
* Switching from dev → prod is just **changing FSPlugin type**.

---

# 🧪 8. Optional Hybrid Mode

* First check **EmbedFS** → if file not found, fallback to **OSFS**
* Example:

```go
func (h *HybridFS) ReadFile(path string) ([]byte, error) {
    data, err := embedFS.ReadFile(path)
    if err == nil { return data, nil }
    return osFS.ReadFile(path)
}
```

* Useful for: hot reload + embedded production binary

---

# ⚡ 9. Advantages

1. Fully **single-binary capable** (all assets embedded)
2. Supports **dev-mode external FS** for hot reload
3. Renderer, router, bundler, CSS, cache all **agnostic to FS type**
4. Clean separation: users just pick FS plugin → rest of pipeline works

---

# 🏗️ 10. Next Steps / LLM Workflow

1. Implement `FSPlugin` interface.
2. Implement **EmbedFSPlugin**, **OSFSPlugin**, optional **HybridFSPlugin**.
3. Update orchestrator + plugins to read **all files via FSPlugin**.
4. Update **hot reload manager** to use FSPlugin.Watch() (OSFS only).
5. Add **config flag** to select FS type at startup.
6. Test both **embedded binary** and **filesystem mode**.

---

This **makes your platform fully single-binary deployable** while preserving:

* Hot reloading in dev mode
* Modular plugin architecture
* Renderer, router, bundler, CSS, cache isolation