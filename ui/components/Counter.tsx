"use client";
// app/components/Counter.tsx
// This is a CLIENT component. It is bundled by esbuild for the browser.
// The Go renderer never executes this file – it only sees a reference object
// (ClientRef) that the browser resolves against the client manifest.

import { useState } from "react";

interface CounterProps {
  initialCount?: number;
  label?: string;
}

export default function Counter({ initialCount = 0, label = "Count" }: CounterProps) {
  const [count, setCount] = useState(initialCount);

  return (
    <div style={{
      display: "inline-flex",
      alignItems: "center",
      gap: "0.75rem",
      background: "#fff",
      border: "1px solid #e0e0e0",
      borderRadius: "8px",
      padding: "0.5rem 1rem",
    }}>
      <span style={{ fontWeight: 600, minWidth: "4rem" }}>{label}: {count}</span>
      <button
        onClick={() => setCount((c) => c - 1)}
        style={{ padding: "0.25rem 0.75rem", borderRadius: "4px", border: "1px solid #ccc", cursor: "pointer" }}
      >
        −
      </button>
      <button
        onClick={() => setCount((c) => c + 1)}
        style={{ padding: "0.25rem 0.75rem", borderRadius: "4px", border: "1px solid #ccc", cursor: "pointer", background: "#0070f3", color: "#fff", borderColor: "#0070f3" }}
      >
        +
      </button>
    </div>
  );
}
