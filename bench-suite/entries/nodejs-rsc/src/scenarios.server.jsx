// Server-Component trees for the Node.js RSC baseline. These run under the
// `react-server` export condition and are serialized to the Flight wire format
// by react-server-dom-webpack/server.node — the SAME protocol Pola uses, so this
// is an apples-to-apples RSC peer for the Pola entries (only the runtime differs:
// native Node V8 here vs Pola's Go-embedded goja/v8go VM).
//
// Content markers match the harness correctness gate and the Pola pages exactly.

import React, { Suspense } from "react";

const delay = (ms) => new Promise((r) => setTimeout(r, ms));

// Scenario 1 — static, no data.
export const scenario1 = () => (
  <main>
    <h1>Benchmark: Static</h1>
    <p>This page renders no data. It measures baseline render cost.</p>
  </main>
);

// Scenario 2 — a Server Component awaits a 50ms source, Suspense-streamed.
async function Async50() {
  await delay(50);
  return <p id="data">Loaded after 50ms</p>;
}
export const scenario2 = () => (
  <main>
    <h1>Benchmark: Async</h1>
    <Suspense fallback={<p id="fallback">Loading…</p>}>
      <Async50 />
    </Suspense>
  </main>
);

// Scenario 4 — nested Suspense, three boundaries at 20/50/200ms.
async function Level({ ms, children }) {
  await delay(ms);
  return (
    <div>
      <p id={`level-${ms}`}>{`Loaded after ${ms}ms`}</p>
      {children}
    </div>
  );
}
export const scenario4 = () => (
  <main>
    <h1>Benchmark: Nested</h1>
    <Suspense fallback={<p id="fallback-20">Loading 20…</p>}>
      <Level ms={20}>
        <Suspense fallback={<p id="fallback-50">Loading 50…</p>}>
          <Level ms={50}>
            <Suspense fallback={<p id="fallback-200">Loading 200…</p>}>
              <Level ms={200} />
            </Suspense>
          </Level>
        </Suspense>
      </Level>
    </Suspense>
  </main>
);

export const scenarios = { 1: scenario1, 2: scenario2, 4: scenario4 };
