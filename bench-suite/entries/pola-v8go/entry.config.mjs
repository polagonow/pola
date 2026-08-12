// Pola (v8go variant: V8/JIT engine + react/RSDW renderer) descriptor.
//
// Same app + same react renderer as pola-goja, but the JS runs in V8 (JIT)
// instead of goja (pure-Go interpreter). This isolates the engine's effect.
//
// Wiring note: engine/v8go shipped without a plugin.go and did not implement the
// current core.SSRPoolFactory / core.SSRRuntime contract. Both were added
// (engine/v8go/plugin.go + a CallRenderFunction/DrainStream/NewSSRPool bridge
// mirroring goja) so the react renderer can drive V8. v8go requires CGO (links
// V8 statically → larger binary).

export default {
  name: "pola-v8go",
  kind: "rsc",
  flight: true,
  install: { cmd: "bash", args: ["setup.sh"] },
  build: { cmd: "bash", args: ["build.sh"], cleanPaths: ["server-bin", "app/public/assets"] },
  start: { cmd: "./server-bin", args: [], env: {} },
  health: "/",
  startTimeoutMs: 40000, // V8 isolate init
  scenarios: {
    1: { path: "/s1" },
    2: { path: "/s2" },
    3: { path: "/s3" },
    4: { path: "/s4" },
  },
  clientBundles: {
    framework: [
      "app/public/assets/_client-*.js",
      "app/public/assets/chunks/*.js",
      "app/public/assets/ErrorBoundary-*.js",
    ],
    app: [
      "app/public/assets/Counter-*.js",
      "app/public/assets/error-*.js",
      "app/public/assets/global-error-*.js",
    ],
  },
  notes: [
    "Pola v8go: V8 (JIT) engine + react renderer (real react-server-dom-webpack). Same client bundle as pola-goja; only the server JS engine differs.",
    "engine/v8go required framework wiring to run at all: added plugin.go + a core.SSRPoolFactory/SSRRuntime bridge (CallRenderFunction/DrainStream/NewSSRPool) mirroring goja. Recorded in FAIRNESS.md.",
    "Requires CGO (CGO_ENABLED=1, --cgo 1); V8 is statically linked → binary ~49 MB vs goja ~17 MB. Not a fully static/scratch-portable binary like goja.",
    "Uses the react renderer, so like pola-goja it does NOT cache Flight by default (Revalidate=0); cache-busting is still applied uniformly.",
    "Two-request model identical to pola-goja.",
  ],
};
