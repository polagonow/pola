import React from "react";

export default function ProjectLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div>
      <a href="/projects" className="text-sm text-[var(--color-muted)] hover:text-[var(--color-fg)] mb-4 inline-block">
        ← All projects
      </a>
      {children}
    </div>
  );
}
