// Generate static SVG charts from results/summary.json into charts/*.svg and a
// CHARTS.md that embeds them. SVGs are self-contained light "cards" (own surface
// background) so they render legibly on GitHub in both light and dark themes.
//
// Design follows the dataviz skill: validated blue/orange categorical palette
// (Pola = blue, Node baseline = orange), ≤24px bars with 4px rounded data-ends
// and 2px surface gaps, recessive hairline grid, text in ink tokens (not series
// color), legend for ≥2 series, selective direct labels. Run: node charts.mjs

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SUMMARY = path.join(__dirname, "results", "summary.json");
const OUT_DIR = path.join(__dirname, "charts");

// ── palette (light surface; validated blue+orange) ─────────────────────────────
const C = {
  surface: "#fcfcfb",
  ink: "#0b0b0b",
  ink2: "#52514e",
  muted: "#898781",
  grid: "#e1e0d9",
  axis: "#c3c2b7",
  pola: "#2a78d6", // series-1 blue
  node: "#eb6834", // series-2 orange
  warn: "#d03b3b", // status critical (for broken markers, with label)
  flaky: "#eda100", // status warning amber (renders, but unreliably)
};

const FONT = 'system-ui, -apple-system, "Segoe UI", sans-serif';

// ── helpers ────────────────────────────────────────────────────────────────────
const esc = (s) => String(s).replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c]);
const num = (n) => (n == null ? "—" : Number(n).toLocaleString("en-US", { maximumFractionDigits: n < 100 ? 1 : 0 }));
const compact = (n) => {
  if (n == null) return "—";
  if (n >= 1000) return (n / 1000).toFixed(n >= 10000 ? 0 : 1) + "k";
  return String(Math.round(n));
};
const groupOf = (name) => (name.startsWith("pola-") ? "pola" : "node");
const colorOf = (name) => (groupOf(name) === "pola" ? C.pola : C.node);

// rounded RIGHT end only (square at baseline/left)
function barPath(x, y, w, h, r) {
  r = Math.min(r, w, h / 2);
  if (w <= r) return `M${x},${y} h${w} v${h} h${-w} Z`;
  return `M${x},${y} h${w - r} a${r},${r} 0 0 1 ${r},${r} v${h - 2 * r} a${r},${r} 0 0 1 ${-r},${r} h${-(w - r)} Z`;
}

function card(bodyW, bodyH, title, subtitle, inner, legend) {
  const padX = 20, padTop = 20, titleH = title ? 44 : 8;
  const W = bodyW + padX * 2;
  const H = bodyH + padTop + titleH + 16;
  const parts = [];
  parts.push(`<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" font-family='${FONT}'>`);
  parts.push(`<rect x="0" y="0" width="${W}" height="${H}" rx="10" fill="${C.surface}"/>`);
  parts.push(`<rect x="0.5" y="0.5" width="${W - 1}" height="${H - 1}" rx="10" fill="none" stroke="rgba(11,11,11,0.10)"/>`);
  if (title) {
    parts.push(`<text x="${padX}" y="${padTop + 6}" font-size="15" font-weight="600" fill="${C.ink}">${esc(title)}</text>`);
    if (subtitle) parts.push(`<text x="${padX}" y="${padTop + 24}" font-size="11.5" fill="${C.muted}">${esc(subtitle)}</text>`);
  }
  if (legend) parts.push(legend(W - padX, padTop + 6));
  parts.push(`<g transform="translate(${padX},${padTop + titleH})">${inner}</g>`);
  parts.push(`</svg>`);
  return parts.join("");
}

function legendSwatches(items) {
  return (rightX, y) => {
    let x = rightX;
    const els = [];
    for (const it of [...items].reverse()) {
      const label = it.label;
      const wText = label.length * 6.2 + 22;
      x -= wText;
      els.push(`<rect x="${x}" y="${y - 8}" width="10" height="10" rx="2" fill="${it.color}"${it.hatch ? ` fill="url(#hatch)"` : ""}/>`);
      els.push(`<text x="${x + 15}" y="${y + 1}" font-size="11" fill="${C.ink2}">${esc(label)}</text>`);
      x -= 12;
    }
    return `<g>${els.join("")}</g>`;
  };
}

// Horizontal bar chart. rows: [{name, value, note, broken}]
function hbar({ title, subtitle, rows, unit, valueFmt = num, refLine, legend = true }) {
  const gutter = 148, barH = 18, gap = 12, rowH = barH + gap;
  const plotW = 560;
  const bodyH = rows.length * rowH + 34;
  const max = Math.max(refLine || 0, ...rows.map((r) => r.value || 0)) * 1.12 || 1;
  const x = (v) => (v / max) * plotW;
  const ticks = axisTicks(max, 4);
  const g = [];
  // gridlines + x ticks
  for (const t of ticks) {
    g.push(`<line x1="${gutter + x(t)}" y1="0" x2="${gutter + x(t)}" y2="${rows.length * rowH}" stroke="${C.grid}"/>`);
    g.push(`<text x="${gutter + x(t)}" y="${rows.length * rowH + 16}" font-size="10" fill="${C.muted}" text-anchor="middle" font-variant-numeric="tabular-nums">${compact(t)}</text>`);
  }
  if (unit) g.push(`<text x="${gutter + plotW}" y="${rows.length * rowH + 30}" font-size="10" fill="${C.muted}" text-anchor="end">${esc(unit)}</text>`);
  if (refLine) {
    g.push(`<line x1="${gutter + x(refLine)}" y1="-4" x2="${gutter + x(refLine)}" y2="${rows.length * rowH + 2}" stroke="${C.warn}" stroke-dasharray="3 3" stroke-width="1"/>`);
    g.push(`<text x="${gutter + x(refLine)}" y="-8" font-size="10" fill="${C.warn}" text-anchor="middle">${esc(String(refLine))}ms source</text>`);
  }
  rows.forEach((r, i) => {
    const y = i * rowH;
    g.push(`<text x="${gutter - 10}" y="${y + barH / 2 + 4}" font-size="11.5" fill="${C.ink}" text-anchor="end">${esc(r.name)}</text>`);
    if (r.broken) {
      // muted hatched stub + status label (no meaningful value)
      g.push(`<path d="${barPath(gutter, y, 8, barH, 3)}" fill="url(#hatch)"/>`);
      g.push(`<text x="${gutter + 16}" y="${y + barH / 2 + 4}" font-size="11" fill="${C.warn}">✗ ${esc(r.note || "excluded")}</text>`);
    } else {
      const w = Math.max(2, x(r.value));
      // flaky = renders correctly but not every time → amber bar + rate label.
      g.push(`<path d="${barPath(gutter, y, w, barH, 4)}" fill="${r.flaky ? C.flaky : colorOf(r.name)}"/>`);
      const lbl = r.flaky ? `${valueFmt(r.value)}  ⚠ flaky ${r.flaky}%` : valueFmt(r.value);
      g.push(`<text x="${gutter + w + 7}" y="${y + barH / 2 + 4}" font-size="11" fill="${r.flaky ? C.flaky : C.ink2}" font-variant-numeric="tabular-nums">${esc(lbl)}</text>`);
    }
  });
  const defs = `<defs><pattern id="hatch" width="6" height="6" patternTransform="rotate(45)" patternUnits="userSpaceOnUse"><rect width="6" height="6" fill="${C.surface}"/><line x1="0" y1="0" x2="0" y2="6" stroke="${C.muted}" stroke-width="2"/></pattern></defs>`;
  const leg = legend
    ? legendSwatches([
        { label: "Pola", color: C.pola },
        { label: "Node baseline", color: C.node },
      ])
    : null;
  return card(gutter + plotW + 60, bodyH, title, subtitle, defs + g.join(""), leg);
}

// Grouped 2-series horizontal bars (cold vs warm build)
function groupedHbar({ title, subtitle, rows, unit }) {
  const gutter = 148, barH = 9, gap = 3, groupGap = 14;
  const rowH = barH * 2 + gap + groupGap, plotW = 540;
  const bodyH = rows.length * rowH + 30;
  const max = Math.max(...rows.flatMap((r) => [r.cold || 0, r.warm || 0])) * 1.14 || 1;
  const x = (v) => (v / max) * plotW;
  const ticks = axisTicks(max, 4);
  const g = [];
  for (const t of ticks) {
    g.push(`<line x1="${gutter + x(t)}" y1="0" x2="${gutter + x(t)}" y2="${rows.length * rowH - groupGap}" stroke="${C.grid}"/>`);
    g.push(`<text x="${gutter + x(t)}" y="${rows.length * rowH - groupGap + 15}" font-size="10" fill="${C.muted}" text-anchor="middle" font-variant-numeric="tabular-nums">${compact(t)}</text>`);
  }
  if (unit) g.push(`<text x="${gutter + plotW}" y="${rows.length * rowH - groupGap + 29}" font-size="10" fill="${C.muted}" text-anchor="end">${esc(unit)}</text>`);
  rows.forEach((r, i) => {
    const y = i * rowH;
    g.push(`<text x="${gutter - 10}" y="${y + barH + 2}" font-size="11.5" fill="${C.ink}" text-anchor="end">${esc(r.name)}</text>`);
    g.push(`<path d="${barPath(gutter, y, Math.max(2, x(r.cold)), barH, 3)}" fill="${C.pola}"/>`);
    g.push(`<text x="${gutter + x(r.cold) + 6}" y="${y + barH - 1}" font-size="9.5" fill="${C.ink2}" font-variant-numeric="tabular-nums">${compact(r.cold)}</text>`);
    g.push(`<path d="${barPath(gutter, y + barH + gap, Math.max(2, x(r.warm)), barH, 3)}" fill="${C.node}"/>`);
    g.push(`<text x="${gutter + x(r.warm) + 6}" y="${y + barH + gap + barH - 1}" font-size="9.5" fill="${C.ink2}" font-variant-numeric="tabular-nums">${compact(r.warm)}</text>`);
  });
  const leg = legendSwatches([
    { label: "cold build", color: C.pola },
    { label: "warm build", color: C.node },
  ]);
  return card(gutter + plotW + 50, bodyH, title, subtitle, g.join(""), leg);
}

// Flush timeline: one lane per entry, dots at chunk arrival ms.
function timeline({ title, subtitle, lanes, marks }) {
  const gutter = 148, laneH = 26, plotW = 540;
  const bodyH = lanes.length * laneH + 34;
  const max = Math.max(...lanes.flatMap((l) => l.points.map((p) => p.tMs)), ...(marks || [])) * 1.08 || 1;
  const x = (v) => (v / max) * plotW;
  const g = [];
  for (const m of marks || []) {
    g.push(`<line x1="${gutter + x(m)}" y1="-4" x2="${gutter + x(m)}" y2="${lanes.length * laneH}" stroke="${C.grid}"/>`);
    g.push(`<text x="${gutter + x(m)}" y="-8" font-size="9.5" fill="${C.muted}" text-anchor="middle">${m}ms</text>`);
  }
  for (const t of axisTicks(max, 5)) {
    g.push(`<text x="${gutter + x(t)}" y="${lanes.length * laneH + 16}" font-size="10" fill="${C.muted}" text-anchor="middle" font-variant-numeric="tabular-nums">${compact(t)}</text>`);
  }
  g.push(`<text x="${gutter + plotW}" y="${lanes.length * laneH + 30}" font-size="10" fill="${C.muted}" text-anchor="end">ms since request</text>`);
  lanes.forEach((l, i) => {
    const y = i * laneH + laneH / 2;
    g.push(`<text x="${gutter - 10}" y="${y + 4}" font-size="11.5" fill="${C.ink}" text-anchor="end">${esc(l.name)}</text>`);
    const xs = l.points.map((p) => gutter + x(p.tMs));
    if (xs.length > 1) g.push(`<line x1="${xs[0]}" y1="${y}" x2="${xs[xs.length - 1]}" y2="${y}" stroke="${colorOf(l.name)}" stroke-opacity="0.35" stroke-width="2"/>`);
    l.points.forEach((p) => {
      g.push(`<circle cx="${gutter + x(p.tMs)}" cy="${y}" r="4.5" fill="${colorOf(l.name)}" stroke="${C.surface}" stroke-width="2"/>`);
    });
  });
  const leg = legendSwatches([
    { label: "Pola", color: C.pola },
    { label: "Node baseline", color: C.node },
  ]);
  return card(gutter + plotW + 50, bodyH, title, subtitle, g.join(""), leg);
}

function axisTicks(max, n) {
  const step = niceStep(max / n);
  const out = [];
  for (let v = 0; v <= max + 1e-9; v += step) out.push(Math.round(v * 1000) / 1000);
  return out;
}
function niceStep(raw) {
  const p = Math.pow(10, Math.floor(Math.log10(raw)));
  const f = raw / p;
  return (f < 1.5 ? 1 : f < 3 ? 2 : f < 7 ? 5 : 10) * p;
}

// ── build charts from summary ──────────────────────────────────────────────────
if (!fs.existsSync(SUMMARY)) {
  console.error("No results/summary.json — run `node bench.mjs` first.");
  process.exit(1);
}
const s = JSON.parse(fs.readFileSync(SUMMARY, "utf8"));
fs.mkdirSync(OUT_DIR, { recursive: true });

const entries = s.entries.filter((e) => !e.fatal);
// scenario ids are stored as strings ("1".."4"); coerce so number args also match.
const sc = (e, id) => e.scenarios?.find((x) => String(x.scenario) === String(id));
const flightOrDoc = (e, id, pick) => {
  const x = sc(e, id);
  if (!x || x.outcome !== "ok") return null;
  const t = x.timings.flightTtlbMs || x.timings.documentTtlbMs;
  return pick === "ttlb" ? t.median : null;
};
const s1load = (e) => {
  const x = sc(e, 1);
  if (!x || x.outcome !== "ok") return null;
  return (x.load.rscFlight || x.load.document)?.requests?.perSec ?? null;
};

const files = [];
function emit(name, svg) {
  fs.writeFileSync(path.join(OUT_DIR, name), svg);
  files.push(name);
}

// 1. Static render throughput (scenario 1)
emit(
  "throughput-static.svg",
  hbar({
    title: "Static render throughput — Scenario 1",
    subtitle: "requests/sec at 50 connections, cache-busted (higher is better)",
    unit: "req/s",
    valueFmt: compact,
    rows: entries
      .map((e) => ({ name: e.name, value: s1load(e) }))
      .filter((r) => r.value != null)
      .sort((a, b) => b.value - a.value),
  }),
);

// 2. Static render latency (scenario 1 TTLB median)
emit(
  "latency-static.svg",
  hbar({
    title: "Static render latency — Scenario 1",
    subtitle: "median time-to-last-byte, ms (lower is better)",
    unit: "ms",
    valueFmt: (v) => num(v) + " ms",
    rows: entries
      .map((e) => ({ name: e.name, value: flightOrDoc(e, 1, "ttlb") }))
      .filter((r) => r.value != null)
      .sort((a, b) => a.value - b.value),
  }),
);

// 3. RSS under sustained load
emit(
  "memory-underload.svg",
  hbar({
    title: "Memory under sustained load",
    subtitle: "peak RSS during a 10s load test, MiB (lower is better)",
    unit: "MiB",
    valueFmt: (v) => num(v),
    rows: entries
      .map((e) => ({ name: e.name, value: e.memoryMiB?.underLoadMax }))
      .filter((r) => r.value != null)
      .sort((a, b) => a.value - b.value),
  }),
);

// 4. Async correctness — scenario 2 flight TTLB vs 50ms source
emit(
  "async-scenario2.svg",
  hbar({
    title: "Async render — Scenario 2 (awaits a 50ms source)",
    subtitle: "median Flight TTLB, ms — a correct engine clears the 50ms line; amber = flaky, ✗ = broken",
    unit: "ms",
    valueFmt: (v) => num(v) + " ms",
    refLine: 50,
    rows: entries.map((e) => {
      const x = sc(e, 2);
      if (x && x.outcome === "ok") {
        return { name: e.name, value: (x.timings.flightTtlbMs || x.timings.documentTtlbMs).median };
      }
      if (x && x.outcome === "flaky") {
        return {
          name: e.name,
          value: (x.timings.flightTtlbMs || x.timings.documentTtlbMs).median,
          flaky: x.correctness ? x.correctness.contentRatePct : null,
        };
      }
      return { name: e.name, broken: true, note: x ? x.outcome : "n/a" };
    }),
  }),
);

// 5. Cold vs warm build
emit(
  "build-time.svg",
  groupedHbar({
    title: "Build time",
    subtitle: "cold (clean) vs warm production build, ms",
    unit: "ms",
    rows: entries
      .map((e) => ({ name: e.name, cold: e.build?.cold?.ms, warm: e.build?.warm?.ms }))
      .filter((r) => r.cold != null),
  }),
);

// 6. Flush timeline — scenario 4 (nested Suspense), working RSC entries
const laneEntries = entries.filter((e) => {
  const x = sc(e, 4);
  return x && x.outcome === "ok" && (x.flushTimeline?.rscFlight?.length || x.flushTimeline?.document?.length);
});
if (laneEntries.length) {
  emit(
    "flush-timeline-s4.svg",
    timeline({
      title: "Streaming timeline — Scenario 4 (nested Suspense 20/50/200ms)",
      subtitle: "each dot = a response chunk flushed; content streams progressively as sources resolve",
      marks: [20, 50, 200, 270],
      lanes: laneEntries.map((e) => {
        const x = sc(e, 4);
        return { name: e.name, points: x.flushTimeline.rscFlight || x.flushTimeline.document };
      }),
    }),
  );
}

// ── CHARTS.md ──────────────────────────────────────────────────────────────────
const env = s.environment;
const md = [];
md.push("# Benchmark charts");
md.push("");
md.push("> Generated by `node charts.mjs` from `results/summary.json`. Do not hand-edit.");
md.push(`> Environment: ${env.cpu.model}, ${env.cpu.logicalCores} cores, ${env.memoryGiB} GiB, ${env.os.version} · Node ${env.runtimes.node} · ${s.config.runs} runs (drop ${s.config.warmupDiscarded}), ${s.config.loadDurationSec}s load.`);
md.push("> **Blue = Pola engines, orange = Node.js baselines.** No winner is declared; read `FAIRNESS.md` for caveats (Pola's two-request model, cache-busting, the async correctness gate).");
md.push("");
const titles = {
  "throughput-static.svg": "Static render throughput",
  "latency-static.svg": "Static render latency",
  "memory-underload.svg": "Memory under load",
  "async-scenario2.svg": "Async correctness (50ms source)",
  "build-time.svg": "Build time",
  "flush-timeline-s4.svg": "Streaming timeline (nested Suspense)",
};
for (const f of files) {
  md.push(`## ${titles[f] || f}`);
  md.push("");
  md.push(`![${titles[f] || f}](charts/${f})`);
  md.push("");
}
fs.writeFileSync(path.join(__dirname, "CHARTS.md"), md.join("\n"));
console.log(`Wrote ${files.length} charts + CHARTS.md`);
