// Index — lists the benchmark scenarios. Not itself a measured scenario.
export default function Home() {
  return (
    <main>
      <h1>Pola benchmark</h1>
      <ul>
        <li><a href="/s1">Scenario 1 — static</a></li>
        <li><a href="/s2">Scenario 2 — async 50ms (streamed)</a></li>
        <li><a href="/s3">Scenario 3 — interactive island</a></li>
        <li><a href="/s4">Scenario 4 — nested Suspense 20/50/200ms</a></li>
      </ul>
    </main>
  );
}
