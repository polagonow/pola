// Pola (quickjsgo engine + react/RSDW renderer) descriptor.
// Pola quickjsgo: QuickJS engine (quickjs-go binding, CGO).
export default {
  name: "pola-quickjsgo",
  kind: "rsc",
  flight: true,
  install: { cmd: "bash", args: ["setup.sh"] },
  build: { cmd: "bash", args: ["build.sh"], cleanPaths: ["server-bin", "app/public/assets"] },
  start: { cmd: "./server-bin", args: [], env: {} },
  health: "/",
  startTimeoutMs: 40000,
  scenarios: { 1: { path: "/s1" }, 2: { path: "/s2" }, 3: { path: "/s3" }, 4: { path: "/s4" } },
  clientBundles: {
    framework: ["app/public/assets/_client-*.js", "app/public/assets/chunks/*.js", "app/public/assets/ErrorBoundary-*.js"],
    app: ["app/public/assets/Counter-*.js", "app/public/assets/error-*.js", "app/public/assets/global-error-*.js"],
  },
  notes: [
    "Pola quickjsgo: QuickJS engine (quickjs-go binding, CGO).",
    "engine/quickjsgo already shipped plugin.go + SSRPoolFactory; requires CGO (--cgo 1).",
    "Same app + react renderer as pola-goja; only the server JS engine differs. Does not cache Flight by default (Revalidate=0); cache-busting applied uniformly.",
  ],
};
