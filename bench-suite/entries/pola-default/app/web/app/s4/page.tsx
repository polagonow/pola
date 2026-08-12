import React, { Suspense } from "react";
import { Source } from "@pola/actions";

// Scenario 4 — nested Suspense with three boundaries resolving at 20/50/200ms.
// Each level awaits its own source, then reveals the next nested boundary, so
// the browser paints progressively as each row streams in over Flight.
async function Level({ ms, children }: { ms: number; children?: React.ReactNode }) {
  const v = await Source.get(ms);
  return (
    <div>
      <p id={`level-${ms}`}>{v!.text}</p>
      {children}
    </div>
  );
}

export default function S4() {
  return (
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
}
