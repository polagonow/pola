// Resident-set-size sampling for a process (macOS/Linux via `ps`).
//
// `ps -o rss= -p <pid>` returns RSS in KiB. We sample on an interval and
// report min/median/max in MiB. NOTE: this samples a single pid; if the entry
// forks worker processes the number under-counts — the orchestrator records
// which entries are single-process so RESULTS.md can flag it.

import { execFile } from "node:child_process";

export function rssKiB(pid) {
  return new Promise((resolve) => {
    execFile("ps", ["-o", "rss=", "-p", String(pid)], (err, stdout) => {
      if (err) return resolve(null);
      const kib = parseInt(String(stdout).trim(), 10);
      resolve(Number.isFinite(kib) ? kib : null);
    });
  });
}

// Sample RSS every `intervalMs` until `stop()` is called. Returns a controller
// with `.stop()` that resolves to { samplesMiB, minMiB, medianMiB, maxMiB }.
export function sampleRSS(pid, intervalMs = 100) {
  const samples = [];
  let running = true;
  const tick = async () => {
    while (running) {
      const kib = await rssKiB(pid);
      if (kib != null) samples.push(kib / 1024);
      await sleep(intervalMs);
    }
  };
  const done = tick();
  return {
    async stop() {
      running = false;
      await done;
      const s = [...samples].sort((a, b) => a - b);
      const med = s.length ? s[Math.floor(s.length / 2)] : null;
      return {
        samplesMiB: samples.map((x) => round(x)),
        minMiB: round(s[0] ?? null),
        medianMiB: round(med),
        maxMiB: round(s[s.length - 1] ?? null),
      };
    },
  };
}

// One-shot RSS in MiB.
export async function rssMiB(pid) {
  const kib = await rssKiB(pid);
  return kib == null ? null : round(kib / 1024);
}

export const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function round(x) {
  return x == null ? null : Math.round(x * 100) / 100;
}
