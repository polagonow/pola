import React from "react";

export default function AboutLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="text-xs font-semibold uppercase tracking-wider text-[var(--color-muted)] mb-4">ℹ About</div>
      {children}
    </div>
  );
}
