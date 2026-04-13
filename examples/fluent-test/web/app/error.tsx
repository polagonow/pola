"use client";
import { useEffect } from "react";

export default function Error({
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
    <div style={{ marginTop: 16, border: "1px solid #d32f2f", borderRadius: 8, padding: 16, background: "#fce4ec", color: "#d32f2f" }}>
      <strong>Something went wrong</strong>
      <p style={{ margin: "8px 0", fontSize: "0.875rem" }}>
        {error.digest ?? error.message}
      </p>
      <button
        onClick={reset}
        style={{ border: "1px solid #d32f2f", background: "transparent", borderRadius: 4, padding: "8px 16px", fontSize: "0.875rem", cursor: "pointer", color: "#d32f2f" }}
      >
        Try again
      </button>
    </div>
  );
}
