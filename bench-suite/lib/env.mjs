// Capture the environment block reported in RESULTS.md.

import os from "node:os";
import { execFileSync } from "node:child_process";

function safe(cmd, args) {
  try {
    return execFileSync(cmd, args, { encoding: "utf8" }).trim();
  } catch {
    return null;
  }
}

export function captureEnv() {
  const cpus = os.cpus();
  let cpuModel = cpus[0]?.model ?? "unknown";
  if (process.platform === "darwin") {
    cpuModel = safe("sysctl", ["-n", "machdep.cpu.brand_string"]) ?? cpuModel;
  }
  return {
    capturedAt: new Date().toISOString(),
    os: {
      platform: process.platform,
      release: os.release(),
      version:
        process.platform === "darwin"
          ? `macOS ${safe("sw_vers", ["-productVersion"]) ?? ""} (${safe("sw_vers", ["-buildVersion"]) ?? ""})`
          : safe("uname", ["-a"]),
      arch: process.arch,
    },
    cpu: {
      model: cpuModel,
      logicalCores: cpus.length,
    },
    memoryGiB: Math.round((os.totalmem() / 1024 ** 3) * 100) / 100,
    runtimes: {
      node: process.version,
      pnpm: safe("pnpm", ["-v"]),
      go: safe("go", ["version"]),
    },
    loadAvg: os.loadavg().map((x) => Math.round(x * 100) / 100),
  };
}
