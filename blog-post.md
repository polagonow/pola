# React Server Components in Go: Why Streaming Is Hard (And How We're Trying to Fix It)

## The Allure of Next.js + Go

If you've spent time in web development circles over the past few years, you've almost certainly heard someone pitch "Next.js frontend + Go backend" as a kind of dream stack. It shows up in conference talks, countless Medium posts, and reportedly even in the infrastructure of some well-known tech companies. And honestly, the appeal is obvious: Go's concurrency model, its near-instant startup times, and its ability to compile to a single binary make it a compelling choice for the server layer, while Next.js handles the React rendering, routing, and SSR story on the frontend.

But here's a question worth asking: **do you really need Next.js in that equation?**

Since Go 1.16 introduced the `embed` package, you can bundle your entire frontend (JS, CSS, HTML, assets) directly into your Go binary. One artifact, no Node.js process in production, no runtime dependency. That's a compelling proposition. So why do teams keep reaching for Next.js?

The answer, it turns out, is more nuanced than it first appears.

---

## The SEO Problem and the Flight Protocol

The primary reason teams still bolt Next.js onto a Go backend is **SEO and the modern React feature set**: specifically, React Server Components (RSC) and streaming SSR.

To understand why this matters, it helps to look at what modern React is actually doing under the hood. The [patterns.dev article on streaming SSR](https://patterns.dev/react/streaming-ssr/) explains it well: traditional SSR renders a full HTML string on the server and sends it in one shot. The browser has to wait for the entire payload before it can start painting anything. With streaming, React can send HTML chunks to the client *as they're generated*, allowing the browser to begin rendering immediately. This meaningfully improves Time to First Byte (TTFB) and First Contentful Paint (FCP).

React 18 formalised this with `renderToPipeableStream` (for Node.js streaming environments) and `renderToReadableStream` (for edge/browser environments). Both integrate with Suspense boundaries: a suspended component holds its slot open in the stream, then flushes its content when data arrives, all within a single HTTP response, no client-side waterfall required.

Beyond streaming HTML, there's the React Server Components model. [As the debugbear.com piece on RSCs covers in depth](https://www.debugbear.com/blog/react-server-components), RSCs run exclusively on the server and are serialized into an intermediate wire format called the **Flight protocol**. It's a line-oriented text format that looks something like this:

```
0:J{"type":"div","props":{"children":"Hello"}}
1:I{"id":"./Button.js","name":"Button","chunks":["chunk-abc"]}
2:H{"href":"/styles.css","as":"style"}
```

Each line is a typed chunk:
- `J`: a JSON-serialized React element tree
- `I`: a client component reference (module + export name)
- `H`: a preload/prefetch hint
- `E`: an error boundary payload

The browser receives this stream, reconstructs the React tree from the chunks, and hydrates only the client components (the `I` entries). Crucially, Suspense works end-to-end: the server emits a placeholder chunk when a component suspends, then flushes the real content as a later chunk when the async work resolves. The client fills in the slot without a full page re-render.

**The key constraint: the Flight protocol requires a streaming transport. You cannot generate it with `renderToString`.**

---

## What the Go Community Has Already Tried

Before going further, a genuine shoutout is owed to several projects that have seriously explored this space. None have cracked streaming yet, but each represents real engineering effort and is worth studying.

### [natewong1313/go-react-ssr](https://github.com/natewong1313/go-react-ssr)

A clean, focused plugin for existing Go web frameworks. It uses esbuild for fast bundling and even auto-generates TypeScript interfaces from Go struct definitions, a nice quality-of-life touch for type safety across the boundary. Under the hood, it leans on `renderToString` from `react-dom/server.browser`, which produces a complete HTML string. Great for basic SSR needs, but no streaming, no server-side Suspense, no Flight.

### [highercomve/go-react-ssr](https://github.com/highercomve/go-react-ssr)

This one takes a different engine approach, using [v8go](https://github.com/rogchap/v8go), a CGO binding to Google's V8 JavaScript engine, to execute React directly on the Go server. The [itnext.io article on rendering React with v8go](https://itnext.io/rendering-react-on-golang-with-v8go-fd84cdad2844) walks through this approach in detail. The appeal is real: you get a genuine V8 context inside your Go process, which means you can run the same `renderToString` call that Node.js would run. No JS-to-Go translation layer, just V8 doing what V8 does. Clever, and it genuinely sidesteps the "Go doesn't run JS" problem. But again: `renderToString` only, no streaming.

### [livebud/bud](https://github.com/livebud/bud)

The most ambitious of the group: a full-stack Go framework with Next.js-style file-based routing conventions, automatic code generation, live reload, and SSR support. It's an impressive piece of engineering and the single-binary dream is fully alive here. But its React rendering is also built on `renderToString`. The streaming story isn't there yet.

### [JLarky/strike](https://github.com/JLarky/strike)

Worth a separate mention because it's the closest thing to a Flight protocol proof of concept in Go. Rather than trying to run `react-server-dom-webpack`, strike implements the RSC wire format manually in Go, serializing React trees to Flight chunks by hand. It works as a proof of concept and it genuinely supports a form of streaming output. But it's a hand-rolled implementation of a protocol that the React team actively evolves, which means it's perpetually one React release away from potential drift. The XSS vulnerabilities the author themselves acknowledge also point to the challenge of reimplementing what React's encoder handles automatically.

---

## The Real Problem: `renderToReadableStream`

All four of the above approaches share the same fundamental constraint. They use `renderToString` (or a manual protocol equivalent), which means:

- No server-side Suspense with data
- No incremental streaming to the client
- No official Flight protocol (or a brittle hand-rolled one)

To unlock real Flight streaming, you need `renderToReadableStream` from `react-server-dom-webpack/server.browser`. This function takes a React server tree, a client component manifest, and returns a **`ReadableStream`** (the browser-native streaming API) that emits Flight protocol chunks as the encoder progresses:

```typescript
import { renderToReadableStream } from "react-server-dom-webpack/server.browser";

const stream = renderToReadableStream(
  <Page {...props} />,
  clientManifest,
  {
    onError(err) { console.error(err); }
  }
);
```

The stream emits `J:` chunks as element trees are serialized, `I:` chunks as client component references are encountered, `H:` hints for preloads, and so on. Suspense boundaries cause the encoder to emit a placeholder, then flush the resolved content as a later chunk. All of this happens over the single stream that gets piped to the HTTP response.

**The problem**: `ReadableStream` doesn't exist in any Go-based JavaScript runtime. Not in [goja](https://github.com/dop251/goja) (a pure-Go ES2020 VM). Not in v8go (the CGO V8 binding). Neither ships Web APIs; they're JavaScript engines, not browser runtimes. So `renderToReadableStream` throws immediately when it tries to construct its output stream.

And it goes deeper. Even if `ReadableStream` were defined, React Flight's serializer relies on a cluster of other Web APIs that aren't part of the ECMAScript spec itself:

- `queueMicrotask`: for scheduling serialization work
- `TextEncoder` / `TextDecoder`: for encoding Flight chunks as UTF-8 bytes
- `MessageChannel`: for internal task scheduling within the encoder
- `AbortController` / `AbortSignal`: for request cancellation

None of these exist in a bare JS engine. They're part of the browser environment (or Node.js, which ships its own implementations). A Go-based JS VM gives you neither.

This is the point where, historically, the answer has been: run a Node.js sidecar. Fire up a Node.js process alongside your Go binary, call it via HTTP or stdio, and accept the operational overhead. It works, but it defeats the whole point of the "single binary" goal. You've traded one dependency (Next.js) for another (Node.js at runtime).

---

## What If We Polyfill the Missing Web APIs?

This is the core question this project is exploring: **can we implement the Web APIs that `renderToReadableStream` needs, in Go, directly on top of a pure-Go JS runtime, well enough to produce real Flight output?**

With AI-assisted development accelerating the spelunking through browser internals, this is an increasingly approachable problem. Here's what the implementation actually looks like.

### Choosing the Runtime: Goja

The project uses [goja](https://github.com/dop251/goja), a pure-Go JavaScript runtime written entirely in Go (no CGO, no V8, no external compilation toolchain). Goja's core is ES5.1 with substantial ES2015+ support added over time (arrow functions, destructuring, Promises, async/await, generators, and more). This matters for a few reasons:

- The compiled binary is truly static and cross-compilable
- No CGO means simpler CI and no platform-specific build concerns
- Go functions can be registered directly as JS globals with zero serialization overhead

For async/await and Promise scheduling, goja is paired with [goja_nodejs](https://github.com/dop251/goja_nodejs) which provides a real event loop. Without this, `async` functions would run to their first `await` and halt; Promise resolution callbacks would never fire.

Polyfills are registered onto the runtime before the server bundle executes, in a specific dependency order:

```go
// vm/goja/polyfill/polyfill.go

// Enable installs all polyfills onto rt as globals.
// Must be called before rt.RunProgram(serverBundle).
func Enable(rt *goja.Runtime) {
	microtask.Enable(rt)       // foundation: must run first
	textencoding.Enable(rt)    // standalone
	messagechannel.Enable(rt)  // depends on __microtaskQueue__
	readablestream.Enable(rt)  // depends on __drainMicrotasks__
	webpackrequire.Enable(rt)  // standalone
	abortcontroller.Enable(rt) // standalone
}
```

Order matters. Microtask is the foundation everything else builds on.

### Polyfill 1: Microtask Queue

React's Flight encoder calls `queueMicrotask()` extensively to schedule serialization work in small batches. In a browser, microtasks drain automatically after each macrotask. In goja, there's no automatic draining; the event loop only processes what we explicitly tell it to.

The polyfill exposes the standard `queueMicrotask` (which React calls) and a Go-accessible `__drainMicrotasks__` function that we invoke at controlled moments to flush all pending work:

```go
// vm/goja/polyfill/microtask/microtask.go

func Enable(rt *goja.Runtime) {
	queue := rt.NewArray()
	rt.Set("__microtaskQueue__", queue)

	pushFn, _ := goja.AssertFunction(queue.Get("push"))

	rt.Set("queueMicrotask", func(call goja.FunctionCall) goja.Value {
		fn := call.Argument(0)
		pushFn(queue, fn)
		return goja.Undefined()
	})

	spliceFn, _ := goja.AssertFunction(queue.Get("splice"))

	rt.Set("__drainMicrotasks__", func(call goja.FunctionCall) goja.Value {
		safety := 0
		for queue.Get("length").ToInteger() > 0 && safety < 5000 {
			safety++
			batchVal, err := spliceFn(queue, rt.ToValue(0))
			if err != nil {
				break
			}
			batch := batchVal.(*goja.Object)
			length := batch.Get("length").ToInteger()
			for i := int64(0); i < length; i++ {
				fn := batch.Get(strconv.FormatInt(i, 10))
				if callable, ok := goja.AssertFunction(fn); ok {
					func() {
						defer func() { recover() }()
						callable(goja.Undefined())
					}()
				}
			}
		}
		return goja.Undefined()
	})
}
```

The safety counter guards against infinite loops. The `recover()` in the inner closure matches JavaScript's try/catch semantics: a bad microtask shouldn't abort the entire drain.

### Polyfill 2: ReadableStream

This is the most critical piece. React's `renderToReadableStream` constructs a `ReadableStream` with an underlying source that has `start` and `pull` methods, the standard WHATWG Streams API. The encoder calls `controller.enqueue(chunk)` to push Flight bytes into the stream as it serializes.

The polyfill installs the standard `ReadableStream` constructor (which React's encoder calls internally) and a Go-accessible `__pullStream__` function that coordinates reading chunks out:

```go
// vm/goja/polyfill/readablestream/readablestream.go

func Enable(rt *goja.Runtime) {
	controllerProto := buildControllerProto(rt)
	streamProto := buildStreamProto(rt)

	rsCtor := rt.ToValue(func(call goja.ConstructorCall) *goja.Object {
		controller := rt.NewObject()
		controller.SetPrototype(controllerProto)
		controller.Set("_chunks", rt.NewArray())
		controller.Set("_closed", rt.ToValue(false))
		controller.Set("_error", goja.Null())

		src := call.Argument(0)
		call.This.Set("_controller", controller)
		call.This.Set("_src", src)
		call.This.Set("_started", rt.ToValue(false))
		call.This.SetPrototype(streamProto)
		return nil
	}).(*goja.Object)
	rt.Set("ReadableStream", rsCtor)

	// __pullStream__: the Go-bridge API called by render.go
	rt.Set("__pullStream__", func(call goja.FunctionCall) goja.Value {
		s := call.Argument(0).ToObject(rt)

		callMethod(rt, s, "_start")

		// Drain microtasks so Flight encoding work completes
		if drain, ok := goja.AssertFunction(rt.Get("__drainMicrotasks__")); ok {
			drain(goja.Undefined())
		}

		callMethod(rt, s, "_pull")

		if drain, ok := goja.AssertFunction(rt.Get("__drainMicrotasks__")); ok {
			drain(goja.Undefined())
		}

		controller := s.Get("_controller").ToObject(rt)
		chunksArr := controller.Get("_chunks").ToObject(rt)
		spliceFn, _ := goja.AssertFunction(chunksArr.Get("splice"))
		chunks, _ := spliceFn(chunksArr, rt.ToValue(0))

		closed := controller.Get("_closed").ToBoolean()
		chunksLen := chunks.ToObject(rt).Get("length").ToInteger()

		result := rt.NewObject()
		result.Set("chunks", chunks)
		result.Set("done", rt.ToValue(closed && chunksLen == 0))
		return result
	})
}
```

The `_start` -> drain -> `_pull` -> drain sequence is deliberate. React's Flight encoder enqueues its serialization work as microtasks during `start` and `pull` calls. If we don't drain between them, the chunks array is empty when we try to splice it. The explicit drain calls are what makes the synchronous polling model work with React's async-by-design encoder.

### Polyfill 3: TextEncoder and TextDecoder

Flight chunks arrive as `Uint8Array` buffers (the encoder uses `TextEncoder.encode()` to serialize strings). To read them back on the Go side, we need `TextDecoder`. Both are straightforward since Go strings are already UTF-8:

```go
// vm/goja/polyfill/textencoding/textencoding.go

proto.Set("encode", func(call goja.FunctionCall) goja.Value {
	s := call.Argument(0).String()
	data := []byte(s) // Go string is always UTF-8
	ab := rt.NewArrayBuffer(data)
	uint8Ctor, _ := goja.AssertConstructor(rt.Get("Uint8Array"))
	obj, _ := uint8Ctor(nil, rt.ToValue(ab))
	return obj
})
```

The `TextDecoder.decode()` side does the inverse: it exports the `Uint8Array` as `[]byte` via goja's reflection, then converts directly to a Go string.

### Polyfill 4: AbortController and MessageChannel

React's streaming encoder uses `AbortController` to signal cancellation (e.g., when the client disconnects) and `MessageChannel` for internal task scheduling within the encoder's event model. Both are implemented as Go functions registered as Goja globals.

`MessageChannel` is the more interesting one: it needs to integrate with the microtask queue so that `port.postMessage()` fires its listener asynchronously in the correct order relative to other microtasks. The implementation wires `postMessage` through `__microtaskQueue__.push()` to maintain the right ordering semantics.

---

## The Rendering Pipeline

With polyfills in place, here's how a request flows through the system:

### 1. Bundle (at startup)

Two esbuild passes happen once when the application starts:

- **Server bundle** (compiled with the `react-server` export condition): includes all page components, layouts, error boundaries, and the `renderToReadableStream` call
- **Client bundle** (ESM): only `"use client"` components, for browser hydration

esbuild runs as a Go package (no Node.js toolchain required, no subprocess).

### 2. VM Pool

The server bundle source is compiled once to a goja `*Program`, then a pool of pre-warmed VMs shares it. Each VM is an independent goja runtime with the program already executed. Per-request state is isolated to a few globals that are cleared on release:

```go
// vm/goja/vm.go

func NewVMPool(serverBundle string, bridge contract.BridgeConfig) (*VMPool, error) {
	prog, err := gojalib.Compile("bundle.js", serverBundle, false)
	if err != nil {
		return nil, fmt.Errorf("vm: compile: %w", err)
	}
	p := &VMPool{bridge: bridge, prog: prog}
	p.pool = sync.Pool{
		New: func() any {
			vm, err := newVM(prog, bridge)
			if err != nil {
				panic(fmt.Sprintf("vm: pool create: %v", err))
			}
			return vm
		},
	}
	// Eagerly create one VM to catch startup errors immediately.
	vm := p.pool.New()
	p.pool.Put(vm)
	return p, nil
}

func (p *VMPool) Acquire() *VM { return p.pool.Get().(*VM) }

func (p *VMPool) Release(vm *VM) {
	_ = vm.run(func(rt *gojalib.Runtime) error {
		rt.Set("__request__", gojalib.Undefined())
		rt.Set("__gojsx_stream__", gojalib.Undefined())
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key)
		}
		return nil
	})
	p.pool.Put(vm)
}
```

VMs are expensive to initialise (the program has to execute once to set up all globals), so pooling amortises that cost across requests.

### 3. Start Render

Go calls `__render__`, a function exported by the generated server entry, passing the page component name and props as JSON. It returns a `ReadableStream` (our polyfill):

```go
// vm/goja/render.go

func StartRender(vm *VM, exportName, propsJSON string) (*RenderSession, error) {
	var sess RenderSession
	err := vm.run(func(rt *gojalib.Runtime) error {
		renderFn, ok := gojalib.AssertFunction(rt.Get("__render__"))
		if !ok {
			return fmt.Errorf("__render__ is not a function")
		}
		sess.PullStreamFn, ok = gojalib.AssertFunction(rt.Get("__pullStream__"))
		if !ok {
			return fmt.Errorf("__pullStream__ is not a function")
		}
		decoderCtor, ok := gojalib.AssertConstructor(rt.Get("TextDecoder"))
		if !ok {
			return fmt.Errorf("TextDecoder is not a constructor")
		}
		var err error
		sess.DecoderObj, err = decoderCtor(rt.NewObject())
		if err != nil {
			return err
		}
		sess.DecodeFn, ok = gojalib.AssertFunction(sess.DecoderObj.Get("decode"))
		if !ok {
			return fmt.Errorf("TextDecoder.decode is not a function")
		}
		sess.Stream, err = renderFn(gojalib.Undefined(), rt.ToValue(exportName), rt.ToValue(propsJSON))
		return err
	})
	return &sess, err
}
```

### 4. Drain Stream

The Go side polls `__pullStream__` in a loop, decoding each batch of `Uint8Array` chunks and writing them directly to the HTTP response:

```go
func DrainStream(vm *VM, w framework.StreamWriter, sess *RenderSession) (wroteAny bool, err error) {
	for {
		var done, noChunks bool
		if err := vm.run(func(rt *gojalib.Runtime) error {
			r, err := sess.PullStreamFn(gojalib.Undefined(), sess.Stream)
			if err != nil {
				return err
			}
			rObj := r.ToObject(rt)
			chunksArr := rObj.Get("chunks").ToObject(rt)
			chunksLen := int(chunksArr.Get("length").ToInteger())
			if chunksLen > 0 {
				var sb strings.Builder
				for i := range chunksLen {
					decoded, err := sess.DecodeFn(sess.DecoderObj, chunksArr.Get(strconv.Itoa(i)))
					if err != nil {
						return err
					}
					sb.WriteString(decoded.String())
				}
				w.WriteRaw([]byte(sb.String()))
				w.Flush()
				wroteAny = true
			} else {
				noChunks = true
			}
			done = rObj.Get("done").ToBoolean()
			return nil
		}); err != nil {
			return wroteAny, err
		}
		if done {
			break
		}
		if noChunks {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return wroteAny, nil
}
```

Each `w.Flush()` call pushes a batch of Flight chunks to the client immediately; that's the actual streaming. The browser's RSC client parses the chunks and incrementally renders the component tree as they arrive.

---

## The JavaScript-Go Interface (JSI)

One of the more interesting design decisions in this project is how Go functions are exposed to React server components. Rather than a generic `fetch`-style API bridge, there's a typed `__JSI__` global that server components call directly:

```typescript
// ui/packages/jsi/index.ts

declare global {
  const __JSI__: {
    getPosts: () => Promise<Post[]>;
    getPost: (slug: string) => Promise<Post>;
    getProjects: () => Promise<Project[]>;
    getProject: (id: string) => Promise<Project>;
    getProfile: (id?: string) => Promise<Profile>;
    getRevisions: (slug: string) => Promise<Revision[]>;
    getRevision: (slug: string, rev: string) => Promise<Revision>;
    triggerError: (message?: string) => Promise<never>;
  };
}

// Per-request context injected by the Go VM before each render.
export declare const __request__: {
  url: string;
  path: string;
  query: string;
  method: string;
  headers: Record<string, string>;
};
```

TypeScript sees these as typed async functions. On the Go side, each function is registered as a closure that immediately returns a Promise, then resolves it asynchronously via the event loop:

```go
// vm/goja/vm.go

func (vm *VM) SetJSI(funcs map[string]contract.GoFunc) error {
	return vm.run(func(rt *gojalib.Runtime) error {
		for name, fn := range funcs {
			fn := fn
			vm.jsi.Set(name, func(c gojalib.FunctionCall) gojalib.Value {
				args := exportArgs(c.Arguments)
				p, resolve, reject := rt.NewPromise()
				go func() {
					result, err := fn(args)
					vm.loop.RunOnLoop(func(rt *gojalib.Runtime) {
						if err != nil {
							reject(rt.ToValue(err.Error()))
						} else {
							resolve(rt.ToValue(result))
						}
					})
				}()
				return rt.ToValue(p)
			})
		}
		return nil
	})
}
```

The Go goroutine runs in parallel with the JS event loop. When the data is ready, `RunOnLoop` schedules the Promise resolution back onto the event loop thread. From the React component's perspective, it's just `await`:

```tsx
export default async function PostsPage() {
  const posts = await __JSI__.getPosts();
  return (
    <ul>
      {posts.map(p => (
        <li key={p.slug}>{p.title}</li>
      ))}
    </ul>
  );
}
```

The Go-side function can hit a database, call another service, read from cache: anything. The server component just awaits the result.

Per-request data (URL, query params, headers) is injected as `__request__` before each render:

```go
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	return vm.run(func(rt *gojalib.Runtime) error {
		rt.Set("__request__", rt.ToValue(ctx))
		return nil
	})
}
```

---

## Is This Actually Achievable?

The question this project started with was: *can you polyfill `ReadableStream` and the supporting Web APIs well enough to run React Flight in a Go-based JS runtime?*

The answer is looking increasingly like **yes**, but with important caveats around fidelity.

React's Flight encoder was written against browser scheduling semantics. Microtasks drain in a specific order. ReadableStream backpressure behaves in specific ways. `MessageChannel` fires at specific points in the task lifecycle. Getting all of those interactions right in a polyfill layer, without a real browser or Node.js, requires understanding what React's encoder actually depends on versus what it merely expects to be there.

The polyfills described above aren't theoretical; they're running, they're producing Flight protocol output, and the output is being consumed by the client-side RSC runtime. That's the meaningful milestone.

What's still being worked through: comprehensive test coverage across React versions, edge cases in stream lifecycle management (what happens when a Suspense boundary rejects? when the client disconnects mid-stream?), and performance profiling under load.

But the architectural insight is solid: **you don't need Node.js to run React Server Components**. You need a JS engine that can execute modern JavaScript, a polyfill layer that implements the subset of Web APIs React's encoder actually uses, and a way to drain that stream into an HTTP response. All of that is achievable in pure Go.

What this could mean if it fully pans out:
- A React Server Components runtime that ships as a **single Go binary**
- No Node.js sidecar, no container with Node pre-installed, no version pinning headaches
- Native Go concurrency for data fetching; goroutines doing the heavy lifting, not JS async/await
- Streaming Flight output from a server written in Go
- A genuine drop-in alternative to the Next.js + Go combo, without the Next.js

If you've worked on similar problems (polyfilling Web APIs for non-browser JS runtimes, or running React in unconventional environments) what challenges did you run into? What would you do differently?
