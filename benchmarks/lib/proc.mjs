// Process orchestration: run build commands, start a server and wait until it
// answers, measure cold-start, and stop it cleanly.

import { spawn } from "node:child_process";
import { sampleRSS, sleep } from "./rss.mjs";
import { measureRequest } from "./measure.mjs";

const nowNs = () => process.hrtime.bigint();
const msSince = (s) => Number(nowNs() - s) / 1e6;

// Run a command to completion. Returns { code, ms, stdout, stderr, peakRssMiB }.
// Samples RSS of the spawned process so we can report peak build memory.
export function runCommand(cmd, args, { cwd, env, label } = {}) {
  return new Promise((resolve) => {
    const startNs = nowNs();
    const child = spawn(cmd, args, {
      cwd,
      env: { ...process.env, ...env },
      stdio: ["ignore", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (d) => (stdout += d));
    child.stderr.on("data", (d) => (stderr += d));
    const rss = sampleRSS(child.pid, 150);
    child.on("error", async (err) => {
      const r = await rss.stop();
      resolve({
        code: -1,
        ms: msSince(startNs),
        stdout,
        stderr: stderr + "\n" + String(err),
        peakRssMiB: r.maxMiB,
        label,
      });
    });
    child.on("exit", async (code) => {
      const r = await rss.stop();
      resolve({ code, ms: msSince(startNs), stdout, stderr, peakRssMiB: r.maxMiB, label });
    });
  });
}

// Start a long-running server. Polls `healthUrl` until it returns a 2xx/3xx/404
// (any HTTP answer counts as "up"), measuring cold-start = spawn → first answer.
// Returns { proc, pid, coldStartMs } or throws with captured logs on failure.
export async function startServer({ cmd, args, cwd, env, healthUrl, timeoutMs = 60000 }) {
  const startNs = nowNs();
  const child = spawn(cmd, args, {
    cwd,
    env: { ...process.env, ...env },
    stdio: ["ignore", "pipe", "pipe"],
  });
  let stdout = "";
  let stderr = "";
  child.stdout.on("data", (d) => (stdout += d));
  child.stderr.on("data", (d) => (stderr += d));

  let exited = false;
  let exitCode = null;
  child.on("exit", (c) => {
    exited = true;
    exitCode = c;
  });

  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (exited) {
      throw new Error(
        `server exited early (code ${exitCode}) before answering.\n--- stdout ---\n${stdout}\n--- stderr ---\n${stderr}`,
      );
    }
    try {
      const res = await measureRequest(healthUrl, { timeoutMs: 2000 });
      if (res.status && res.status < 500) {
        return { proc: child, pid: child.pid, coldStartMs: round(msSince(startNs)), logs: () => ({ stdout, stderr }) };
      }
    } catch {
      // not up yet
    }
    await sleep(120);
  }
  child.kill("SIGKILL");
  throw new Error(
    `server did not become healthy within ${timeoutMs}ms.\n--- stdout ---\n${stdout}\n--- stderr ---\n${stderr}`,
  );
}

export async function stopServer(proc) {
  if (!proc || proc.exitCode !== null) return;
  proc.kill("SIGTERM");
  const gone = await Promise.race([
    new Promise((r) => proc.on("exit", () => r(true))),
    sleep(4000).then(() => false),
  ]);
  if (!gone) proc.kill("SIGKILL");
}

function round(x) {
  return x == null ? null : Math.round(x * 1000) / 1000;
}
