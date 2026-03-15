"use client"
import { useEffect } from "react";

export default function Error({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => { console.error(error); }, [error]);
  return (
    <div className="card rsc-err" style={{ marginTop: "1rem" }}>
      <strong>Something went wrong</strong>
      <p style={{ fontSize: ".9rem", margin: ".5rem 0" }}>{error.message}</p>
      <button className="btn btn-outline" onClick={reset}>Try again</button>
    </div>
  );
}
