// Scenario 1 — static page, no data. Content matches the control's canonical DOM.
export default function S1() {
  return (
    <main>
      <h1>Benchmark: Static</h1>
      <p>This page renders no data. It measures baseline render cost.</p>
    </main>
  );
}
