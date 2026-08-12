// Pola (nativersc variant: goja engine + Go-native Flight renderer) descriptor.
//
// Same app source as pola-default, built with `--renderer nativersc`. Instead of
// running Meta's react-server-dom-webpack inside the VM, nativersc walks the
// React element tree and serializes the Flight wire format in Go
// (renderer/nativersc/flight.go, reconciler.go). Same two-request model and the
// same client runtime (@pola/react/client), so the harness measures it exactly
// like pola-default.

export default {
  name: "pola-nativersc",
  kind: "rsc",
  flight: true,
  install: { cmd: "bash", args: ["setup.sh"] },
  build: { cmd: "bash", args: ["build.sh"], cleanPaths: ["server-bin", "app/public/assets"] },
  start: { cmd: "./server-bin", args: [], env: {} },
  health: "/",
  startTimeoutMs: 30000,
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
    "Pola nativersc: goja engine + Go-native Flight renderer (renderer/nativersc). The RSC payload is serialized in Go, not by react-server-dom-webpack in the VM.",
    "Flight requests are detected by `Content-Type: text/x-component` only (nativersc.go:106) — the harness sends both Content-Type and Accept, matching the real Pola client.",
    "Two-request model identical to pola-default: `document` rows measure the shell, `RSC Flight` rows carry render cost.",
    "Client runtime is the same @pola/react/client bundle as pola-default; client JS framework/app split is the same filename heuristic.",
    "CSRF + security headers ON (Pola production defaults).",
  ],
};
