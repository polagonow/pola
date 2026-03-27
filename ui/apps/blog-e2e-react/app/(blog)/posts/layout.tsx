import React from "react";

export default function PostsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="text-xs font-semibold uppercase tracking-wider text-[var(--color-muted)] mb-4">✍︎ Blog</div>
      {children}
    </div>
  );
}
