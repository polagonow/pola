// Control entry descriptor consumed by the benchmark orchestrator.

export default {
  name: "control",
  kind: "ssr", // plain streaming SSR, no RSC
  install: { cmd: "pnpm", args: ["install"] },
  build: { cmd: "node", args: ["build.mjs"], cleanPaths: ["dist"] },
  start: { cmd: "node", args: ["server.mjs"], env: { NODE_ENV: "production" } },
  health: "/health",
  startTimeoutMs: 20000,
  scenarios: {
    1: { path: "/s1" },
    2: { path: "/s2" },
    3: { path: "/s3" },
    4: null, // RSC-only scenario → N/A for a non-RSC control
  },
  flight: null, // not an RSC entry; single-request model
  clientBundles: { all: ["dist/client/*.js"] },
  clientReportFile: "dist/client/report.json",
  notes: [
    "Raw react-dom/server renderToPipeableStream; no framework. The floor.",
    "Scenario 3 hydrates the whole document root — plain React has no partial hydration without RSC; the interactive 'island' is one Counter in an otherwise static tree.",
    "Scenario 4 (nested RSC Suspense) is N/A for a non-RSC control, per the plan.",
    "No gzip/brotli on the wire (raw); the harness computes gzip/brotli sizes offline from response bodies.",
  ],
};
