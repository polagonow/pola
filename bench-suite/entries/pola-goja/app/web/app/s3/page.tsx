import { Counter } from "../../components/Counter";

// Scenario 3 — one interactive Client Component island in an otherwise server tree.
export default function S3() {
  return (
    <main>
      <h1>Benchmark: Island</h1>
      <p>A static server tree wrapping one interactive island.</p>
      <Counter />
    </main>
  );
}
