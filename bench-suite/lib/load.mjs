// Load test via autocannon (programmatic). Reports latency percentiles,
// throughput, and requests/sec. Returns null-shaped result on failure so a
// single entry's problems don't abort the whole run.

import { createRequire } from "node:module";
const require = createRequire(import.meta.url);

export async function loadTest(url, { headers = {}, connections = 50, duration = 10 } = {}) {
  let autocannon;
  try {
    autocannon = require("autocannon");
  } catch {
    return { error: "autocannon not installed (run `pnpm install` in bench-suite/)" };
  }
  return new Promise((resolve) => {
    const instance = autocannon(
      { url, connections, duration, headers, excludeErrorStats: false },
      (err, result) => {
        if (err) return resolve({ error: String(err) });
        resolve({
          requests: {
            total: result.requests.total,
            perSec: round(result.requests.average),
          },
          latencyMs: {
            mean: round(result.latency.mean),
            p50: round(result.latency.p50),
            p95: round(result.latency.p97_5 ?? result.latency.p95),
            p99: round(result.latency.p99),
            max: round(result.latency.max),
          },
          throughputBytesPerSec: round(result.throughput.average),
          errors: result.errors,
          non2xx: result.non2xx,
          timeouts: result.timeouts,
          connections,
          durationSec: duration,
        });
      },
    );
    // Surface progress-free; no track() to keep stdout clean during orchestration.
    instance.on("error", (e) => resolve({ error: String(e) }));
  });
}

function round(x) {
  return x == null ? null : Math.round(x * 100) / 100;
}
