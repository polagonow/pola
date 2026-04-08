"use client";
import { useEffect } from "react";

export default function PostsError({
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
    <div className="text-red-600 bg-red-50 px-4 py-3 rounded-lg border border-red-200">
      <strong>Failed to load posts</strong>
      <p className="text-[var(--color-muted)] text-sm my-2">
        {error.digest ?? error.message}
      </p>
      <button className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-transparent text-[var(--color-fg)] border-[var(--color-border)] hover:bg-[var(--color-surface)] hover:no-underline" onClick={reset}>
        Try again
      </button>
    </div>
  );
}
