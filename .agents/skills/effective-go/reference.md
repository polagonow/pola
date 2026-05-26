## Effective Go — comprehensive rule paraphrase

This document is a **non-verbatim**, section-by-section **paraphrase** of the guidance on the Effective Go page. When in doubt, treat the canonical page as the source of truth: `https://go.dev/doc/effective_go`.

### Scope note (from the page)

- **Context**: The document was written around Go’s early releases and is not a complete guide to newer ecosystem features (modules, modern tooling, generics, etc.).
- **How to use it**: Use it as an idioms/style guide for the language and standard patterns; combine with modern docs when dealing with newer ecosystem topics.

---

## Formatting

- **Use gofmt**: Let `gofmt` (or `go fmt`) standardize indentation and alignment, including comment alignment in structs.
- **Indentation**: Tabs are the default indentation; use spaces only when you must.
- **Line length**: There is no strict line length limit; wrap when a line feels too long and use sensible indentation.
- **Parentheses**: Go typically needs fewer parentheses than C/Java; control statements do not use parentheses around conditions.
- **Operator precedence**: It’s simpler than in many languages; prefer spacing that makes intent obvious.

## Commentary (comments)

- **Prefer line comments** (`//`) for most cases; use block comments (`/* ... */`) mainly for package comments or when useful inside expressions.
- **Doc comments**: A comment immediately preceding a top-level declaration (with no blank line) documents that declaration and is used by `go doc` tooling.
- **Write comments for users**: Comments should improve understanding where code alone doesn’t.

## Names

### Package names

- **Keep package names short**: lower-case, single word, no underscores, no mixedCaps.
- **Name packages for how clients use them**: the import name becomes the qualifier (`bytes.Buffer`), so keep it clear and concise.
- **Avoid redundant exported names**: exported identifiers can omit the package name because call sites already qualify them (`bufio.Reader`, not `bufio.BufReader`).
- **Directory base name sets package name**: e.g. import path `encoding/base64` has package `base64`.
- **Avoid `import .`**: it can be useful in certain testing scenarios but should otherwise be avoided because it obscures provenance.

### Getters and setters

- **Avoid “Get” prefixes**: if the field is `owner`, the exported getter should be `Owner()`, not `GetOwner()`.
- **Setters**: If you need one, `SetOwner` is common and reads well.

### Interface names

- **One-method interfaces**: typically method name + `-er` (`Reader`, `Writer`, `Formatter`), or a close variation.
- **Honor standard meanings**: don’t name a method `Read`, `Write`, `String`, `Close`, etc. unless it matches the conventional signature/meaning.
- **Prefer consistency**: If your method has the same semantics as a well-known one, use the same name/signature.

### MixedCaps

- **Identifiers**: Use `MixedCaps`/`mixedCaps` for multiword names, not underscores.

## Semicolons

- **Semicolons are mostly implicit**: the lexer inserts them at end-of-statement tokens, so you almost never write them.
- **Brace placement rule**: you cannot put the opening `{` of `if/for/switch/select` on the next line (semicolon insertion would change meaning). Put `{` on the same line as the control header.
- **When you do use semicolons**: mainly inside `for init; cond; post` clauses or multiple statements on a single line (rare).

---

## Control structures

### If

- **Braces are mandatory**; multi-line `if` is idiomatic.
- **Use `if` with an init statement** when it improves locality (e.g., `if err := ...; err != nil { ... }`).
- **Prefer early returns**: if the `if` block ends the control path (`return`, `break`, `continue`, `goto`), omit the `else` branch and let the happy path flow downward.

### Redeclaration and reassignment with `:=`

- **`:=` can reassign**: within the same scope, `:=` may reuse existing variables as long as at least one new variable is introduced and the assigned value is compatible.
- **Common pattern**: reusing a single `err` through successive operations.
- **Watch scope carefully**: parameters and named return values share the function body scope; avoid shadowing surprises.

### For

- **One looping construct**: Go’s `for` covers traditional `for`, `while`, and infinite loops.
- **Common forms**:
  - `for init; cond; post { ... }`
  - `for cond { ... }`
  - `for { ... }`
- **Use short declarations** when helpful (`for i := 0; ...`).
- **Use `range`** for arrays/slices/strings/maps/channels.
- **Drop unused `range` results**:
  - only key/index: `for k := range m { ... }`
  - only value: `for _, v := range xs { ... }`
- **Strings and `range`**: iterates over Unicode code points (runes), decoding UTF-8; invalid encoding yields the replacement rune.
- **No comma operator**; `++`/`--` are statements (not expressions).
- **Multiple loop variables**: use parallel assignment in the post clause (`i, j = i+1, j-1`).

### Switch

- **More general than C**: cases can be non-constant and non-integers; evaluation is top-to-bottom.
- **Expression-less switch**: `switch { ... }` is a common replacement for `if/else if` ladders.
- **No automatic fallthrough**: use `fallthrough` only when you explicitly intend it (and understand it).
- **Comma-separated cases**: allow multi-value match in a single case.
- **Breaking out of outer loops**: use labeled `break`/`continue` when needed; labels apply to loops (and `break` can target a label).

### Type switch

- **Discover dynamic types**: `switch v := i.(type) { ... }` branches by the concrete type of an interface value.
- **Variable reuse pattern**: it’s idiomatic to reuse the same variable name with a new, more specific type in each case.

---

## Functions

### Multiple return values

- **Return (value, error)** is idiomatic for operations that may fail.
- **Use multi-values to express results** without exceptions; don’t encode error states into sentinel values when `error` is appropriate.

### Named result parameters

- **Use sparingly**: named returns can improve clarity for long functions or when return values are semantically important.
- **Avoid overuse**: named returns can hide what’s returned; keep returns explicit when it improves readability.

### Defer

- **Use `defer` to simplify cleanup**: close files, unlock mutexes, etc., near where the resource is acquired.
- **Defer runs in LIFO order** at function return.
- **Defer arguments evaluate immediately**: the function call is deferred, but arguments are captured at the time `defer` executes.

---

## Data

### Allocation with `new` and `make`

- **`new(T)`**: allocates zeroed `T` and returns `*T`.
- **`make`**: initializes slices, maps, and channels (and returns the value, not a pointer) because these types have runtime descriptors that need initialization.
- **Choose by intent**: `new` for plain allocation; `make` for initializing these built-in reference-like types.

### Arrays

- **Arrays are values** with fixed length; passing an array copies it.
- **Use slices more often**: arrays are useful for fixed-size semantics and sometimes for performance or API constraints.

### Slices

- **Slices describe segments** of arrays (pointer + length + capacity); they share underlying storage.
- **Appending may reallocate**: `append` can allocate a new underlying array when capacity is exceeded; avoid relying on underlying sharing unless intentional.
- **Copying**: use `copy(dst, src)` for slice copying.
- **Nil vs empty**: `nil` slice is valid and often preferred as the zero value; treat it consistently.

### Two-dimensional slices

- **Representation choices**:
  - slice of slices (flexible row lengths)
  - single backing array with slicing into rows (contiguous memory; can be faster)
- **Pick based on access patterns** and memory behavior.

### Maps

- **Maps are reference-like**: use `make(map[K]V)` to initialize.
- **Lookup yields (value, ok)**: use the “comma ok” idiom to test presence.
- **Zero value**: a nil map can be read from but cannot be assigned into.
- **Iteration order is unspecified**: don’t rely on `range` iteration order over maps.

### Printing

- **Prefer `fmt`** formatting verbs; use `%v`, `%+v`, `%#v` appropriately.
- **Implement `String()`** for user-friendly representations when it improves debugging/UX.

### Append

- **Prefer `append`** over manual indexing growth logic.
- **Understand capacity behavior**: preallocate when it matters, but don’t prematurely optimize.

---

## Initialization

### Constants

- **Use constants** for values known at compile time; take advantage of `iota` when it improves clarity.
- **Keep constant groups readable**; avoid cleverness that obscures meaning.

### Variables

- **Prefer short variable declarations** when the type is obvious from the initializer.
- **Use explicit types** when it clarifies intent or prevents accidental type inference issues.

### The `init` function

- **Use `init()` for setup** that can’t be expressed as declarations.
- **Avoid overusing `init()`**: keep initialization explicit and testable; prefer constructors or explicit setup where possible.
- **Multiple init functions** per package/file are allowed but can reduce clarity if abused.

---

## Methods

### Pointers vs values

- **Receiver choice matters**:
  - Use pointer receivers to mutate the receiver, avoid copying large structs, or when consistency across methods matters.
  - Value receivers can be fine for small, immutable-like types.
- **Consistency**: choose one receiver style per type when reasonable; mixed styles can surprise users.

### Interfaces and methods

- **Implement interfaces implicitly**: don’t declare intent; just implement the methods.
- **Prefer small interfaces**: split broad interfaces into focused ones when that makes APIs easier to satisfy and test.
- **Design for use sites**: define interfaces where they’re consumed, not necessarily where they’re implemented.

---

## The blank identifier (`_`)

- **Discard unused values**: use `_` to ignore values you intentionally don’t need.
- **Avoid unused-variable workarounds**: don’t bind values just to silence the compiler; structure code so unused values aren’t produced/bound.
- **Multiple return values**: `_` is useful when you only need a subset of returned values.

---

## Embedding

- **Composition is the Go default**: embed types to reuse behavior and optionally promote methods/fields.
- **Promoted methods become part of the outer type’s method set**; this affects interfaces and API surface.
- **Use embedding to improve the API**, not just to avoid typing or to mimic inheritance.
- **Be mindful of conflicts**: name collisions and ambiguity can harm readability.

---

## Concurrency

### Share memory by communicating

- **Prefer channels** for coordinating ownership of data; avoid shared mutable state when channels express the design better.
- **Use goroutines** for concurrent work; keep lifetimes bounded and cancellation explicit when needed.

### Goroutines

- **Lightweight**: spawning goroutines is cheap, but not free; avoid leaks by ensuring goroutines can exit.
- **Capture loop variables carefully**: when launching goroutines in loops, ensure the goroutine gets the intended variable value (e.g., pass as parameter).

### Channels

- **Channels coordinate**: use them to pass values and signal events.
- **Directionality**: use `chan<- T` / `<-chan T` in APIs to express intent.
- **Buffered vs unbuffered**: unbuffered enforces synchronization; buffered can decouple producers/consumers—pick based on correctness, then performance.
- **Closing**: close channels from the sending side to signal completion; receivers should handle the “ok” value when ranging/receiving.

### Select

- **Multiplex**: `select` waits on multiple channel operations.
- **Default case**: use to avoid blocking (but be careful—can create busy loops).
- **Timeouts**: combine `select` with timers/tickers for time-based behavior.

---

## Errors

- **Errors are values**: return them, pass them, and handle them explicitly.
- **Add context**: when returning an error, include enough information to diagnose the failing operation.
- **Avoid overusing panic** for ordinary failure; prefer errors for expected problems.
- **Sentinel and typed errors**: use approaches that allow callers to branch on expected conditions (modern Go typically uses `errors.Is/As`, but the core idea is: make errors actionable).

---

## Panic and recover

- **Panic is for truly unrecoverable conditions** (or programmer errors), not routine error handling.
- **Recover only at appropriate boundaries**: use it to prevent a whole program/server from crashing, but keep behavior predictable and logged.
- **Keep invariants**: if you recover, ensure you restore invariants or fail safely.

---

## A (small) web server example (how to read it)

- **Treat it as a pattern demonstration**:
  - simple handler functions
  - composing behavior with types/methods
  - leveraging interfaces
  - using concurrency primitives and error handling idioms where needed

---

## Practical “apply everywhere” distilled rules

- **Let tools format**: always gofmt.
- **Name for call sites**: short, mixedCaps, avoid redundancy and “Get” prefixes.
- **Prefer early returns**: keep the happy path visible.
- **Use `range`, `switch`, `_`**: bind only what you use; pick constructs that reduce noise.
- **Use `make` for slices/maps/chans**; understand slice sharing/capacity.
- **Keep interfaces small**; define them near consumers.
- **Compose with embedding carefully**; don’t simulate inheritance.
- **Use goroutines/channels deliberately**; avoid leaks; close channels from senders.
- **Return errors, don’t panic** (except for invariants/programmer errors); add useful context.

