import React from "react";

export default function ProfileLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="text-xs font-semibold uppercase tracking-wider text-[var(--color-muted)] mb-4">👤 Profile</div>
      {children}
    </div>
  );
}
