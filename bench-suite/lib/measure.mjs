// Single-request streaming measurement.
//
// Records, for one HTTP GET:
//   - ttfbMs        time from request start to the FIRST body byte
//   - headersMs     time to response headers
//   - ttlbMs        time to the last body byte (response 'end')
//   - flush         [{ tMs, bytes }] one entry per received chunk, in order,
//                   so out-of-order / progressive streaming is visible
//   - totalBytes    body size on the wire (decoded of transfer-encoding, but
//                   NOT of content-encoding — see `body` for the raw buffer)
//   - status, headers
//   - body          Buffer of the full response body (for size/gzip/brotli)
//
// Timings use process.hrtime.bigint() → milliseconds (float).

import http from "node:http";
import https from "node:https";

const nowNs = () => process.hrtime.bigint();
const msSince = (startNs) => Number(nowNs() - startNs) / 1e6;

export function measureRequest(url, { headers = {}, timeoutMs = 30000 } = {}) {
  return new Promise((resolve, reject) => {
    const u = new URL(url);
    const mod = u.protocol === "https:" ? https : http;
    const startNs = nowNs();
    const flush = [];
    const chunks = [];
    let headersMs = null;
    let ttfbMs = null;
    let total = 0;

    const req = mod.request(
      u,
      {
        method: "GET",
        headers: { "accept-encoding": "identity", ...headers },
      },
      (res) => {
        headersMs = msSince(startNs);
        res.on("data", (chunk) => {
          const t = msSince(startNs);
          if (ttfbMs === null) ttfbMs = t;
          total += chunk.length;
          flush.push({ tMs: round(t), bytes: chunk.length });
          chunks.push(chunk);
        });
        res.on("end", () => {
          resolve({
            status: res.statusCode,
            headers: res.headers,
            headersMs: round(headersMs),
            ttfbMs: round(ttfbMs ?? headersMs),
            ttlbMs: round(msSince(startNs)),
            flushCount: flush.length,
            flush,
            totalBytes: total,
            body: Buffer.concat(chunks),
          });
        });
      },
    );

    req.setTimeout(timeoutMs, () => {
      req.destroy(new Error(`request timeout after ${timeoutMs}ms: ${url}`));
    });
    req.on("error", reject);
    req.end();
  });
}

function round(x, dp = 3) {
  if (x === null || x === undefined || !Number.isFinite(x)) return x;
  const f = 10 ** dp;
  return Math.round(x * f) / f;
}
