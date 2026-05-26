---
name: effective-go
description: Apply Effective Go idioms for clear, idiomatic Go code. Use when writing or refactoring Go (.go) code, reviewing Go changes, or when the user mentions "Effective Go", idiomatic Go, naming, formatting, interfaces, methods, embedding, slices/maps, concurrency, or control-flow style.
---

# Effective Go (skill)

Use this skill to apply the *idioms* from [Effective Go](https://go.dev/doc/effective_go) while writing or reviewing Go code. Prefer the simplest Go solution that reads cleanly to other Go developers.

For a comprehensive, section-by-section paraphrase of **all** guidance from the page, see [`reference.md`](reference.md).

## Defaults (apply unless a strong reason)

- **Format with gofmt**: Don’t hand-align; let `gofmt`/`go fmt` decide indentation/alignment.
- **Name things for how they’re used**:
  - Package names: short, lower-case, single word; avoid underscores and mixedCaps in package names.
  - Exported names avoid redundancy with the package name (`bufio.Reader`, not `bufio.BufReader`).
  - Avoid `Get` in getters (`Owner()`, not `GetOwner()`); setters can be `SetOwner`.
  - One-method interfaces are usually `Method` + `-er` (`Reader`, `Writer`).
  - Use `MixedCaps`/`mixedCaps` for multiword identifiers.
- **Keep control flow flat (“happy path down the page”)**:
  - Handle error/guard cases early and return; omit `else` if the `if` branch returns/breaks/continues.
  - Use `if init; condition { ... }` when it improves locality.
- **Use the blank identifier intentionally**:
  - In `range` loops, use `_` to ignore values you don’t need; don’t bind variables you won’t use.
- **Prefer `switch` over long `if-else` chains** when it improves clarity:
  - No implicit fallthrough; cases can be comma-separated.
  - Use `switch { ... }` (boolean switch) to replace repetitive conditions.
- **Keep interfaces small**:
  - Prefer small, purpose-specific interfaces; accept interfaces, return concrete types when appropriate.
- **Prefer composition/embedding over inheritance**:
  - Embed to reuse implementation/promote methods when it improves the API.
  - Don’t embed merely to “save typing” if it obscures ownership/behavior.
- **Be deliberate with `:=`**:
  - It can reassign already-declared variables in the same scope if at least one new variable is declared; avoid accidental shadowing (especially `err`).

## Review checklist (Effective Go lens)

- **Formatting**: Would `gofmt` change it? If yes, don’t fight it.
- **Naming**: Are names short, consistent, MixedCaps, and non-redundant with package/type context?
- **Flow**: Does the successful path read top-to-bottom with early returns on errors/guards?
- **Loops**: Does `range` bind only needed values? Any unused variables?
- **Switches**: Would a `switch` read better than `if-else`?
- **Interfaces**: Are interfaces minimal and behavior-named? Are we accepting interfaces where helpful?
- **Composition**: Are we embedding/structuring types to make the API clearer (not just shorter)?

