"use client";
// app/components/ThemeToggle.tsx

import { useState, useEffect } from "react";

export default function ThemeToggle() {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    document.body.style.background = dark ? "#111" : "#f8f8f8";
    document.body.style.color = dark ? "#f0f0f0" : "#111";
  }, [dark]);

  return (
    <button
      onClick={() => setDark((d) => !d)}
      style={{
        padding: "0.4rem 1rem",
        borderRadius: "6px",
        border: "1px solid #ccc",
        cursor: "pointer",
        background: dark ? "#333" : "#fff",
        color: dark ? "#fff" : "#111",
        fontWeight: 500,
      }}
    >
      {dark ? "☀️ Light" : "🌙 Dark"}
    </button>
  );
}
