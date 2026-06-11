"use client";

import { useTransition } from "react";
import { csrfToken } from "@/utils/csrf";

export default function LogoutButton() {
  const [isPending, startTransition] = useTransition();

  function handleLogout() {
    startTransition(async () => {
      await fetch("/logout", {
        method: "POST",
        headers: { "X-CSRF-Token": csrfToken() },
      });
      window.location.href = "/";
    });
  }

  return (
    <button onClick={handleLogout} disabled={isPending} className="px-4 py-2 border border-[var(--color-border)] rounded-lg text-sm font-medium cursor-pointer bg-transparent text-[var(--color-fg)] disabled:opacity-60">
      {isPending ? "Logging out..." : "Logout"}
    </button>
  );
}
