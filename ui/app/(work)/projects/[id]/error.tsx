"use client"
import { useEffect } from "react";

export default function ProjectError({ error, reset }: { error: Error; reset: () => void }) {
  useEffect(() => { console.error(error); }, [error]);
  return (
    <div className="card" style={{ borderColor: "#fecaca" }}>
      <strong style={{ color: "#dc2626" }}>Failed to load project</strong>
      <p style={{ color: "var(--muted)", fontSize: ".9rem", margin: ".5rem 0" }}>{error.message}</p>
      <button className="btn btn-outline" onClick={reset}>Try again</button>
    </div>
  );
}
