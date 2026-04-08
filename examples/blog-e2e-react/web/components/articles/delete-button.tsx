"use client";

import { useTransition } from "react";
import { csrfToken } from "@/utils/csrf";

export default function DeleteButton({ id }: { id: number }) {
  const [isPending, startTransition] = useTransition();

  function handleDelete() {
    if (!window.confirm("Are you sure you want to delete this article?")) {
      return;
    }

    startTransition(async () => {
      try {
        const res = await fetch(`/articles/${id}`, {
          method: "DELETE",
          headers: { "X-CSRF-Token": csrfToken() },
        });

        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || res.statusText);
        }

        window.location.href = "/articles";
      } catch (err) {
        alert(err instanceof Error ? err.message : "Delete failed");
      }
    });
  }

  return (
    <button
      onClick={handleDelete}
      disabled={isPending}
      style={{
        background: "none",
        border: "none",
        color: "#dc2626",
        cursor: isPending ? "not-allowed" : "pointer",
        fontSize: "inherit",
        padding: 0,
        opacity: isPending ? 0.6 : 1,
      }}
    >
      {isPending ? "Deleting..." : "Delete"}
    </button>
  );
}