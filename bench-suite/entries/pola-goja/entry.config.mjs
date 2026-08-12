// Pola (default: goja engine + react/RSDW renderer) entry descriptor.
//
// Pola uses the two-request model: a normal GET returns a shell; the content
// streams over a second request with `Accept: text/x-component`. Setting
// kind:"rsc" + flight:true makes the harness measure BOTH requests.

export default {
  name: "pola-goja",
  kind: "rsc",
  flight: true, // enable two-request (document + RSC Flight) measurement
  // Frontend deps install; the Go binary build happens in the build step.
  install: { cmd: "bash", args: ["setup.sh"] },
  // Regenerate the JS bridge + two-stage `pola build` → ./server-bin.
  // NOTE: Go's compiler cache is process-global and is NOT cleared here, so for
  // Pola "cold" vs "warm" build differ mainly in JS bundling + linking, not full
  // Go recompilation. Recorded in FAIRNESS.md.
  build: { cmd: "bash", args: ["build.sh"], cleanPaths: ["server-bin", "app/public/assets"] },
  start: { cmd: "./server-bin", args: [], env: {} },
  health: "/",
  startTimeoutMs: 30000,
  scenarios: {
    1: { path: "/s1" },
    2: { path: "/s2" },
    3: { path: "/s3" },
    4: { path: "/s4" }, // RSC-only nested Suspense — supported here
  },
  // Client JS shipped to the browser. Framework runtime vs app split is a
  // filename heuristic for Pola (no metafile attribution like the control):
  //   framework = Pola client runtime + React chunks + framework error boundary
  //   app       = the scenario's own client components (Counter) + route error UIs
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
    "Default Pola: goja (pure-Go interpreter, no JIT) + react renderer (real react-server-dom-webpack).",
    "Two-request model: `document` rows measure the shell; `RSC Flight` rows measure the streamed payload. TTFB of the document is shell-only and not comparable to SSR entries' content TTFB — compare the Flight request for render cost.",
    "Client JS framework/app split is a filename heuristic (not exact metafile attribution); the Pola client runtime bundles React + the Flight parser into _client-*.js.",
    "CSRF + security headers are ON (Pola production defaults) — GET scenarios are unaffected; recorded in FAIRNESS.md.",
    "Every page is wrapped in a Suspense boundary whose fallback is loading.tsx (framework default), in addition to scenario-specific boundaries.",
  ],
};
