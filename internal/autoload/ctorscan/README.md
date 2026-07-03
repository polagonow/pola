# ctorscan

Shared AST scanner used by the `repos`, `services`, `routes`, and `actionbridge`
autoloaders to enforce and describe Pola's constructor convention:

```go
func NewFoo(r *core.Registry) *Foo { … }   // canonical
func NewFoo()               *Foo { … }     // also allowed
```

That's it. Any other parameter — `*gorm.DB`, `repositories.FooRepository`, a
config struct, a service interface — is rejected at generation time with a
diagnostic that names the file, constructor, and parameter index. Dependencies
must be resolved from the DI registry inside the constructor body:

```go
func NewFoo(r *core.Registry) *Foo {
    return &Foo{
        db:   core.MustInvoke[*gorm.DB](r),
        repo: core.MustInvoke[repositories.FooRepository](r),
    }
}
```

## Why one signature

The framework already resolves ORM handles (`*gorm.DB`, `*entsql.Driver`,
`orm.Ormer`), the ent client (via the auto-generated `EntClientPlugin`), the
web framework, storage, cache, session, MCP server, and every user
repository/service through `core.Registry`. Everything a user constructor might
need is either already there or will be by the time the constructor runs. A
positional parameter list only re-encodes information the registry already
knows, and it forces every autoloader (repos, services, routes, actions) to
implement its own "resolve then pass" plumbing. Anchoring on a single
`(r *core.Registry)` signature makes autoloaders trivial:

```go
return NewFoo(r), nil
```

…and puts the "which dependencies?" decision inside the constructor body, where
the code that actually cares about them lives. It also makes the rules
identical across concepts — the MCP autoload has always worked this way and now
so does everything else.

## API

```go
// Param is one constructor parameter, resolved for code generation.
type Param struct {
    IsRegistry bool    // true iff the type is *<polaPackage>/core.Registry
    Type       string  // e.g. "*core.Registry"
    Import     *Import // nil for registry params and same-package idents
}

// Import identifies a package that must be imported by the generated file.
type Import struct { Alias, Path string }

// ScanParams walks fd's parameter list, resolving qualifiers via file.Imports.
// Returns an error when a param isn't *core.Registry, when the file uses a dot
// import that hides the qualifier, or when the syntactic shape isn't a named
// type (funcs, channels, generic instantiations, anonymous structs).
func ScanParams(fd *ast.FuncDecl, file *ast.File, filename, polaPackage string) ([]Param, error)

// MergeImports deduplicates a slice of Params' imports for rendering into a
// single generated file. Kept as a utility even though the strict single-param
// rule means most generators only ever see zero or one import.
func MergeImports(params []Param, skip map[string]struct{}) (imports []Import, renames map[string]string)
```

The `*core.Registry` detection matches on the *resolved* import path, so
aliased imports work transparently:

```go
import pc "github.com/polagonow/pola/core"

func NewFoo(r *pc.Registry) *Foo { … }   // still flagged IsRegistry
```

A value parameter (`core.Registry` without the `*`) is rejected with a
diagnostic that suggests `*core.Registry`. A dot import is rejected because it
makes bare identifiers ambiguous.

## Diagnostics

Callers wrap the error with their own context. Typical output:

```
autoload repos: repositories/gorm/todo_repository.go: NewTodoRepository parameter 1 has type *gorm.DB; constructors must take (r *core.Registry) or no parameters — resolve dependencies from the registry via core.MustInvoke inside the constructor body
```

## Where it's used

| Autoloader                          | What it scans                                | What it emits                                                                 |
|-------------------------------------|----------------------------------------------|-------------------------------------------------------------------------------|
| `internal/autoload/repos`           | `repositories/{orm}/New*Repository`          | `pola_plugins.go` with a `*Plugin()` per repo, plus `EntClientPlugin` for ent |
| `internal/autoload/services`        | `services/New*Service`                       | `pola_plugins.go` with a `*Plugin()` per service (+ interface alias provider) |
| `internal/autoload/routes`          | `routes/**/NewRoute`                         | Per-package `pola_route_init.go` calling `routes.Register`                    |
| `internal/actionbridge` (via parser)| `actions/New<Struct>`                        | `generated_bridge.go` mapping every action method into the JS runtime         |

Each autoloader passes `ScanParams` the constructor's `*ast.FuncDecl`, the file
containing it, its filename (for diagnostics), and `PluginOpts.PolaPackage`
(usually `github.com/polagonow/pola`).
