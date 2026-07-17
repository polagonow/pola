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
    <div className="mt-4 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-destructive">
      <strong>Something went wrong</strong>
      <p className="my-2 text-sm">
        {error.digest ?? error.message}
      </p>
      <button
        onClick={reset}
        className="rounded-md border bg-transparent px-4 py-2 text-sm cursor-pointer hover:bg-muted"
      >
        Try again
      </button>
    </div>
  );
}
