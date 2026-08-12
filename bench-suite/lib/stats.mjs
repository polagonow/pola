// Descriptive statistics for benchmark samples.
// All inputs are arrays of numbers (already warmup-trimmed by the caller).

export function median(xs) {
  return percentile(xs, 50);
}

export function percentile(xs, p) {
  if (xs.length === 0) return null;
  const s = [...xs].sort((a, b) => a - b);
  if (p <= 0) return s[0];
  if (p >= 100) return s[s.length - 1];
  // Nearest-rank method (deterministic, no interpolation surprises).
  const rank = Math.ceil((p / 100) * s.length);
  return s[Math.min(rank, s.length) - 1];
}

export function mean(xs) {
  if (xs.length === 0) return null;
  return xs.reduce((a, b) => a + b, 0) / xs.length;
}

export function stddev(xs) {
  if (xs.length < 2) return 0;
  const m = mean(xs);
  const v = xs.reduce((a, b) => a + (b - m) ** 2, 0) / (xs.length - 1);
  return Math.sqrt(v);
}

// Coefficient of variation as a percentage (stddev / mean * 100).
export function cv(xs) {
  const m = mean(xs);
  if (m === null || m === 0) return null;
  return (stddev(xs) / m) * 100;
}

// Summarize a sample array into the shape RESULTS.md reports.
export function summarize(xs) {
  const clean = xs.filter((x) => typeof x === "number" && Number.isFinite(x));
  return {
    n: clean.length,
    median: round(median(clean)),
    p95: round(percentile(clean, 95)),
    p99: round(percentile(clean, 99)),
    min: round(clean.length ? Math.min(...clean) : null),
    max: round(clean.length ? Math.max(...clean) : null),
    cvPct: round(cv(clean)),
  };
}

export function round(x, dp = 3) {
  if (x === null || x === undefined || !Number.isFinite(x)) return null;
  const f = 10 ** dp;
  return Math.round(x * f) / f;
}
