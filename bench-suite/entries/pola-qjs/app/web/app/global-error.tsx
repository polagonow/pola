"use client";
import { useEffect } from "react";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <div style={{
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
      minHeight: "100vh",
      flexDirection: "column",
      gap: "1rem",
    }}>
      <h1 style={{ fontSize: "1.5rem", fontWeight: 700 }}>Something went wrong</h1>
      <p style={{ color: "var(--color-muted)" }}>
        {error.digest ?? error.message}
      </p>
      <button
        onClick={reset}
        style={{
          padding: "0.5rem 1rem",
          borderRadius: "0.5rem",
          border: "1px solid var(--color-border)",
          background: "transparent",
          cursor: "pointer",
        }}
      >
        Try again
      </button>
    </div>
  );
}
