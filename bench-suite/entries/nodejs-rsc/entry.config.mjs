// Node.js RSC baseline: react-server-dom-webpack on native Node V8, same Flight
// protocol and two-request model as Pola. This is the apples-to-apples RSC peer
// for the Pola entries — only the runtime differs (native Node vs Pola's embedded
// Go JS VM).

export default {
  name: "nodejs-rsc",
  kind: "rsc",
  flight: true,
  install: { cmd: "pnpm", args: ["install"] },
  build: { cmd: "node", args: ["build.mjs"], cleanPaths: ["dist"] },
  // Server Components require the react-server export condition.
  start: {
    cmd: "node",
    args: ["--conditions", "react-server", "server.mjs"],
    env: { NODE_ENV: "production" },
  },
  health: "/health",
  startTimeoutMs: 20000,
  scenarios: {
    1: { path: "/s1" },
    2: { path: "/s2" },
    // Scenario 3 (client-component island) needs the "use client" bundler
    // machinery (client references + manifest + chunk loading) that a framework
    // provides — Pola/Next do, a raw Node RSC baseline does not. Marked N/A
    // rather than approximated.
    3: null,
    4: { path: "/s4" },
  },
  clientBundles: { all: ["dist/client/*.js"] },
  clientReportFile: "dist/client/report.json",
  notes: [
    "Node.js RSC baseline: react-server-dom-webpack/server.node under --conditions react-server; same Flight wire protocol Pola uses.",
    "Apples-to-apples RSC peer for the Pola entries — isolates the runtime (native Node V8 vs Pola's embedded goja/v8go VM + Go plumbing).",
    "Two-request model matches Pola: `document` rows measure the shell, `RSC Flight` rows carry the render cost.",
    "Scenario 3 (client island) is N/A: standalone client-reference/manifest wiring is exactly the framework machinery Pola provides and a raw Node baseline lacks.",
    "No gzip/brotli on the wire (raw); the harness computes gzip/brotli offline.",
  ],
};
