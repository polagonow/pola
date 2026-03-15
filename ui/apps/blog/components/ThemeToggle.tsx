"use client"
import { useEffect, useState } from "react";

export default function ThemeToggle() {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    const root = document.documentElement;
    root.style.setProperty("--bg",      dark ? "#0f172a" : "#fff");
    root.style.setProperty("--fg",      dark ? "#f1f5f9" : "#111");
    root.style.setProperty("--surface", dark ? "#1e293b" : "#f9fafb");
    root.style.setProperty("--border",  dark ? "#334155" : "#e5e7eb");
    root.style.setProperty("--muted",   dark ? "#94a3b8" : "#6b7280");
  }, [dark]);

  return (
    <button className="btn btn-ghost" onClick={() => setDark(d => !d)} aria-label="Toggle theme">
      {dark ? "☀︎" : "☾"}
    </button>
  );
}
