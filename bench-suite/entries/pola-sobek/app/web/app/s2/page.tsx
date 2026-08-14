import React, { Suspense } from "react";
import { Source } from "@pola/actions";

// Scenario 2 — a Server Component awaits a 50ms source; the shell streams first
// with the fallback, then the resolved content streams over Flight.
async function Data() {
  const v = await Source.get(50);
  return <p id="data">{v!.text}</p>;
}

export default function S2() {
  return (
    <main>
      <h1>Benchmark: Async</h1>
      <Suspense fallback={<p id="fallback">Loading…</p>}>
        <Data />
      </Suspense>
    </main>
  );
}
