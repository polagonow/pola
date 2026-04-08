"use client";
import { useEffect, useState } from "react";

export default function ThemeToggle() {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    const root = document.documentElement;
    root.style.setProperty("--color-bg", dark ? "#0f172a" : "#fff");
    root.style.setProperty("--color-fg", dark ? "#f1f5f9" : "#111");
    root.style.setProperty("--color-surface", dark ? "#1e293b" : "#f9fafb");
    root.style.setProperty("--color-border", dark ? "#334155" : "#e5e7eb");
    root.style.setProperty("--color-muted", dark ? "#94a3b8" : "#6b7280");
  }, [dark]);

  return (
    <button
      className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-transparent text-[var(--color-muted)] hover:bg-[var(--color-surface)] hover:text-[var(--color-fg)] hover:no-underline"
      onClick={() => setDark((d) => !d)}
      aria-label="Toggle theme"
    >
      {dark ? "☀︎" : "☾"}
    </button>
  );
}
