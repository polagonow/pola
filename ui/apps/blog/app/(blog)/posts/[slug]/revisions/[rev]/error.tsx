"use client"
import { useEffect } from "react";

export default function RevisionError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => { console.error(error); }, [error]);
  return (
    <div className="card rsc-err">
      <strong>Could not load revision</strong>
      <p style={{ fontSize: ".9rem", margin: ".5rem 0" }}>{error.digest ?? error.message}</p>
      <button className="btn btn-outline" onClick={reset}>Try again</button>
    </div>
  );
}
