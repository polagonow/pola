// Benchmark orchestrator.
//
// For each selected entry, one at a time (no concurrency, no shared node_modules):
//   install → cold build → warm build → start → per-scenario measurement → stop.
//
// Every failure (build error, unsupported scenario, crash) is recorded as that
// OUTCOME in the results JSON — never estimated, never silently skipped.
//
// Usage:
//   node bench.mjs                       # all entries, full run
//   node bench.mjs --only control        # one entry
//   node bench.mjs --only control,pola-default
//   node bench.mjs --scenarios 1,2       # subset of scenarios
//   node bench.mjs --runs 7 --warmup 2 --load 10 --connections 50
//   node bench.mjs --quick               # runs=3 warmup=1 load=3 (dev)
//   node bench.mjs --skip-install --skip-build   # reuse prior build artifacts

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { captureEnv } from "./lib/env.mjs";
import { runCommand, startServer, stopServer } from "./lib/proc.mjs";
import { measureRequest } from "./lib/measure.mjs";
import { loadTest } from "./lib/load.mjs";
import { sampleRSS, rssMiB, sleep } from "./lib/rss.mjs";
import { sizeReport, bundleSize } from "./lib/sizes.mjs";
import { summarize } from "./lib/stats.mjs";
import { compareScenario } from "./lib/conformance.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ENTRIES_DIR = path.join(__dirname, "entries");
const RESULTS_DIR = path.join(__dirname, "results");
const PORT = 41234;
const BASE = `http://127.0.0.1:${PORT}`;
// The Pola browser client requests Flight with `Content-Type: text/x-component`
// (see docs/ssr-caching.md). The default `react` renderer also accepts an
// `Accept` header, but `nativersc` keys only off Content-Type — so we send both,
// matching a real client and satisfying every renderer.
const FLIGHT_MIME = "text/x-component";
const FLIGHT_HEADERS = { "content-type": FLIGHT_MIME, accept: FLIGHT_MIME };

function parseArgs(argv) {
  const a = {
    only: null,
    scenarios: ["1", "2", "3", "4"],
    runs: 7,
    warmup: 2,
    load: 10,
    connections: 50,
    skipInstall: false,
    skipBuild: false,
    conformanceOnly: false,
  };
  for (let i = 2; i < argv.length; i++) {
    const arg = argv[i];
    const next = () => argv[++i];
    if (arg === "--only") a.only = next().split(",").map((s) => s.trim());
    else if (arg === "--scenarios") a.scenarios = next().split(",").map((s) => s.trim());
    else if (arg === "--runs") a.runs = +next();
    else if (arg === "--warmup") a.warmup = +next();
    else if (arg === "--load") a.load = +next();
    else if (arg === "--connections") a.connections = +next();
    else if (arg === "--skip-install") a.skipInstall = true;
    else if (arg === "--skip-build") a.skipBuild = true;
    else if (arg === "--conformance-only") a.conformanceOnly = true;
    else if (arg === "--quick") {
      a.runs = 3;
      a.warmup = 1;
      a.load = 3;
    }
  }
  return a;
}

async function loadEntries(only) {
  if (!fs.existsSync(ENTRIES_DIR)) return [];
  const names = fs
    .readdirSync(ENTRIES_DIR)
    .filter((n) => fs.existsSync(path.join(ENTRIES_DIR, n, "entry.config.mjs")));
  const selected = only ? names.filter((n) => only.includes(n)) : names;
  const entries = [];
  for (const name of selected.sort()) {
    const cfgPath = path.join(ENTRIES_DIR, name, "entry.config.mjs");
    const mod = await import(pathToFileUrl(cfgPath));
    entries.push({ ...mod.default, dir: path.join(ENTRIES_DIR, name) });
  }
  return entries;
}

function pathToFileUrl(p) {
  return "file://" + p;
}

async function measureScenario(entry, scenarioId, args) {
  const sc = entry.scenarios?.[scenarioId];
  if (!sc) {
    return { scenario: scenarioId, outcome: "N/A", reason: sc === null ? "not applicable for this entry" : "not implemented" };
  }
  const isRSC = entry.kind === "rsc" && entry.flight;
  const flightHeaders = isRSC ? FLIGHT_HEADERS : null;
  // Cache-bust every measured request with a unique query so we measure true
  // RENDER cost, not server-side full-response cache hits. Pages ignore
  // searchParams, so output is byte-identical; applied uniformly to every entry.
  // (Pola's nativersc renderer caches Flight responses by default — TTL=0/forever
  // — so without this its numbers would be cache-hit latency, not render latency.
  // Default caching behavior is documented separately in FAIRNESS.md.)
  let bustN = 0;
  const bust = () => `${BASE}${sc.path}?__bench=${scenarioId}_${bustN++}`;
  const url = BASE + sc.path; // for logging/reference only

  // --- streaming samples (doc request, and flight request for RSC) ---
  const docTtfb = [];
  const docTtlb = [];
  const flTtfb = [];
  const flTtlb = [];
  let repDocFlush = null;
  let repFlightFlush = null;
  let docBody = null;
  let flightBody = null;
  let firstError = null;

  const total = args.runs;
  for (let i = 0; i < total; i++) {
    try {
      const doc = await measureRequest(bust());
      if (i >= args.warmup) {
        docTtfb.push(doc.ttfbMs);
        docTtlb.push(doc.ttlbMs);
      }
      // keep the representative flush timeline from the last kept run
      if (i === total - 1) {
        repDocFlush = doc.flush;
        docBody = doc.body;
      }
      if (isRSC) {
        const fl = await measureRequest(bust(), { headers: flightHeaders });
        if (i >= args.warmup) {
          flTtfb.push(fl.ttfbMs);
          flTtlb.push(fl.ttlbMs);
        }
        if (i === total - 1) {
          repFlightFlush = fl.flush;
          flightBody = fl.body;
        }
      }
    } catch (e) {
      firstError ??= String(e);
    }
    await sleep(60);
  }

  if (docBody == null && firstError) {
    return { scenario: scenarioId, outcome: "error", reason: firstError };
  }

  // --- payload sizes ---
  // For RSC the "RSC payload" is the flight body; the doc is just the shell.
  const payload = {
    document: sizeReport(docBody ?? Buffer.alloc(0)),
  };
  if (isRSC && flightBody) payload.rscFlight = sizeReport(flightBody);

  // --- load test (the content-producing request), cache-busted uniformly ---
  const load = {
    document: await loadTest(url, { connections: args.connections, duration: args.load, cacheBust: true }),
  };
  if (isRSC) {
    load.rscFlight = await loadTest(url, {
      headers: FLIGHT_HEADERS,
      connections: args.connections,
      duration: args.load,
      cacheBust: true,
    });
  }

  // conformance capture: SSR -> doc HTML; RSC -> needs browser (flagged)
  const conformanceHtml = isRSC ? null : (docBody ? docBody.toString("utf8") : null);

  return {
    scenario: scenarioId,
    outcome: "ok",
    path: sc.path,
    kind: entry.kind,
    twoRequest: !!isRSC,
    timings: {
      documentTtfbMs: summarize(docTtfb),
      documentTtlbMs: summarize(docTtlb),
      ...(isRSC
        ? { flightTtfbMs: summarize(flTtfb), flightTtlbMs: summarize(flTtlb) }
        : {}),
    },
    flushTimeline: {
      document: repDocFlush,
      ...(isRSC ? { rscFlight: repFlightFlush } : {}),
    },
    payloadBytes: payload,
    load,
    _conformanceHtml: conformanceHtml,
    _conformanceNote: isRSC
      ? "RSC entry: first response is a shell; DOM conformance requires browser-rendered capture (Playwright) — pending."
      : null,
  };
}

async function benchEntry(entry, args) {
  const out = {
    name: entry.name,
    kind: entry.kind,
    dir: path.relative(__dirname, entry.dir),
    build: {},
    server: {},
    memoryMiB: {},
    clientJsBytes: null,
    scenarios: [],
    notes: entry.notes ?? [],
  };

  // 1. install
  if (!args.skipInstall && entry.install) {
    log(`  [${entry.name}] install: ${entry.install.cmd} ${entry.install.args.join(" ")}`);
    const r = await runCommand(entry.install.cmd, entry.install.args, {
      cwd: entry.dir,
      env: entry.install.env,
      label: "install",
    });
    out.build.install = { ok: r.code === 0, ms: round(r.ms) };
    if (r.code !== 0) {
      out.build.outcome = "install-failed";
      out.build.stderr = tail(r.stderr);
      return out;
    }
  }

  // 2. cold build (clean caches first) + peak RSS
  if (!args.skipBuild && entry.build) {
    for (const p of entry.build.cleanPaths ?? []) {
      fs.rmSync(path.join(entry.dir, p), { recursive: true, force: true });
    }
    log(`  [${entry.name}] cold build`);
    const cold = await runCommand(entry.build.cmd, entry.build.args, {
      cwd: entry.dir,
      env: entry.build.env,
      label: "cold-build",
    });
    out.build.cold = { ok: cold.code === 0, ms: round(cold.ms), peakRssMiB: cold.peakRssMiB };
    if (cold.code !== 0) {
      out.build.outcome = "build-failed";
      out.build.stderr = tail(cold.stderr);
      return out;
    }
    // 3. warm build (no clean)
    log(`  [${entry.name}] warm build`);
    const warm = await runCommand(entry.build.cmd, entry.build.args, {
      cwd: entry.dir,
      env: entry.build.env,
      label: "warm-build",
    });
    out.build.warm = { ok: warm.code === 0, ms: round(warm.ms) };
  }

  // client JS bytes (total gzip/brotli per group, + raw framework/app split)
  if (entry.clientBundles) {
    out.clientJsBytes = {};
    for (const [group, patterns] of Object.entries(entry.clientBundles)) {
      out.clientJsBytes[group] = bundleSize(entry.dir, patterns);
    }
  }
  if (entry.clientReportFile) {
    const rp = path.join(entry.dir, entry.clientReportFile);
    if (fs.existsSync(rp)) {
      try {
        out.clientJsBytes ??= {};
        out.clientJsBytes.attributionRawBytes = JSON.parse(fs.readFileSync(rp, "utf8"));
      } catch {
        /* ignore malformed report */
      }
    }
  }

  // 4. start server (cold start = spawn -> first successful response)
  // Resolve a relative launcher (e.g. "./server-bin") against the entry dir —
  // child_process.spawn otherwise resolves it against the harness cwd.
  let startCmd = entry.start.cmd;
  if (startCmd.startsWith("./") || startCmd.startsWith("../")) {
    startCmd = path.resolve(entry.dir, startCmd);
  }
  let started;
  try {
    started = await startServer({
      cmd: startCmd,
      args: entry.start.args,
      cwd: entry.dir,
      env: { ...entry.start.env, PORT: String(PORT) },
      healthUrl: BASE + (entry.health ?? "/"),
      timeoutMs: entry.startTimeoutMs ?? 60000,
    });
    out.server.coldStartMs = started.coldStartMs;
  } catch (e) {
    out.server.outcome = "start-failed";
    out.server.error = tail(String(e));
    return out;
  }

  try {
    // idle RSS (let it settle)
    await sleep(500);
    out.memoryMiB.idle = await rssMiB(started.pid);

    // sustained-load RSS: sample while hammering scenario 1 (or first available)
    const loadScenario = entry.scenarios["1"] ?? Object.values(entry.scenarios).find(Boolean);
    if (loadScenario) {
      const rss = sampleRSS(started.pid, 100);
      await loadTest(BASE + loadScenario.path, {
        cacheBust: true,
        connections: args.connections,
        duration: Math.min(args.load, 5),
        headers: entry.kind === "rsc" && entry.flight ? FLIGHT_HEADERS : {},
      });
      const r = await rss.stop();
      out.memoryMiB.underLoadMax = r.maxMiB;
      out.memoryMiB.underLoadMedian = r.medianMiB;
    }

    // per-scenario measurement
    for (const id of args.scenarios) {
      log(`  [${entry.name}] scenario ${id}`);
      const res = await measureScenario(entry, id, args);
      out.scenarios.push(res);
    }
  } finally {
    await stopServer(started.proc);
  }

  return out;
}

function summarizeOutcome(entryResult) {
  if (entryResult.build?.outcome) return entryResult.build.outcome;
  if (entryResult.server?.outcome) return entryResult.server.outcome;
  const oks = entryResult.scenarios.filter((s) => s.outcome === "ok").length;
  return `ok (${oks}/${entryResult.scenarios.length} scenarios)`;
}

async function main() {
  const args = parseArgs(process.argv);
  const entries = await loadEntries(args.only);
  fs.mkdirSync(RESULTS_DIR, { recursive: true });

  const env = captureEnv();
  fs.writeFileSync(path.join(RESULTS_DIR, "environment.json"), JSON.stringify(env, null, 2));

  if (entries.length === 0) {
    log("No entries found under entries/*/entry.config.mjs. Nothing to run.");
    log("(Build an entry directory with an entry.config.mjs to measure it.)");
    return;
  }

  log(`Environment: ${env.cpu.model}, ${env.cpu.logicalCores} cores, ${env.memoryGiB} GiB, ${env.os.version}`);
  log(`Node ${env.runtimes.node} | ${env.runtimes.go ?? "go: n/a"}`);
  log(`Entries: ${entries.map((e) => e.name).join(", ")}`);
  log(`Config: runs=${args.runs} warmup=${args.warmup} load=${args.load}s connections=${args.connections}\n`);

  const allResults = [];
  for (const entry of entries) {
    log(`=== ${entry.name} (${entry.kind}) ===`);
    let result;
    try {
      result = await benchEntry(entry, args);
    } catch (e) {
      result = { name: entry.name, kind: entry.kind, fatal: tail(String(e)) };
    }
    result.outcome = result.fatal ? "fatal" : summarizeOutcome(result);
    allResults.push(result);
    fs.writeFileSync(
      path.join(RESULTS_DIR, `entry-${entry.name}.json`),
      JSON.stringify(result, null, 2),
    );
    log(`    outcome: ${result.outcome}\n`);
  }

  // cross-entry conformance per scenario (SSR entries; RSC pending browser)
  const conformance = {};
  for (const id of args.scenarios) {
    const htmlByEntry = {};
    const pending = [];
    for (const r of allResults) {
      const sc = r.scenarios?.find((s) => s.scenario === id);
      if (!sc) continue;
      if (sc._conformanceHtml) htmlByEntry[r.name] = sc._conformanceHtml;
      else if (sc._conformanceNote) pending.push(r.name);
    }
    conformance[id] = {
      ...compareScenario(htmlByEntry),
      comparedEntries: Object.keys(htmlByEntry),
      pendingBrowserCapture: pending,
    };
  }

  // strip bulky private fields before writing the summary
  for (const r of allResults) {
    for (const s of r.scenarios ?? []) {
      delete s._conformanceHtml;
    }
  }

  const summary = {
    environment: env,
    config: {
      runs: args.runs,
      warmupDiscarded: args.warmup,
      loadDurationSec: args.load,
      connections: args.connections,
    },
    conformance,
    entries: allResults,
  };
  fs.writeFileSync(path.join(RESULTS_DIR, "summary.json"), JSON.stringify(summary, null, 2));

  log("Conformance by scenario:");
  for (const [id, c] of Object.entries(conformance)) {
    const cmp = c.comparedEntries.length;
    const verdict = cmp <= 1 ? "n/a (need ≥2 SSR entries)" : c.conformant ? "PASS" : "FAIL";
    log(`  scenario ${id}: ${verdict} (compared ${cmp}: ${c.comparedEntries.join(", ") || "none"}${c.pendingBrowserCapture.length ? "; pending browser: " + c.pendingBrowserCapture.join(", ") : ""})`);
  }
  log(`\nWrote results/summary.json (${allResults.length} entries).`);
}

function log(m) {
  process.stdout.write(m + "\n");
}
function round(x) {
  return x == null ? null : Math.round(x * 1000) / 1000;
}
function tail(s, n = 2000) {
  s = String(s ?? "");
  return s.length > n ? s.slice(-n) : s;
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
