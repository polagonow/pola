// Shared component tree for the control entry.
//
// The visible DOM produced here is the CANONICAL content for each scenario —
// every other entry (Pola, Next.js, React Router) must render equivalent
// normalized DOM to pass the conformance gate.

import React, { use, useState } from "react";

export function Document({ title, children, bootstrap }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <title>{title}</title>
      </head>
      <body>
        <div id="root">{children}</div>
      </body>
    </html>
  );
}

// Scenario 1 — static page, no data.
export function StaticBody() {
  return (
    <main>
      <h1>Benchmark: Static</h1>
      <p>This page renders no data. It measures baseline render cost.</p>
    </main>
  );
}

// Scenario 2 — awaits a 50ms source, Suspense-streamed.
export function AsyncBody({ dataPromise }) {
  const data = use(dataPromise);
  return (
    <main>
      <h1>Benchmark: Async</h1>
      <p id="data">{data}</p>
    </main>
  );
}

export function AsyncFallback() {
  return <p id="fallback">Loading…</p>;
}

// Scenario 3 — one interactive island in an otherwise static tree.
export function Counter() {
  const [n, setN] = useState(0);
  return (
    <button id="counter" type="button" onClick={() => setN(n + 1)}>
      Count: {n}
    </button>
  );
}

export function IslandBody() {
  return (
    <main>
      <h1>Benchmark: Island</h1>
      <p>A static server tree wrapping one interactive island.</p>
      <Counter />
    </main>
  );
}
