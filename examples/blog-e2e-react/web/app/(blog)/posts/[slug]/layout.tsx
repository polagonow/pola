import React from "react";

export default function PostLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <article>
      <a href="/posts" className="text-sm text-[var(--color-muted)] hover:text-[var(--color-fg)] mb-4 inline-block">
        ← All posts
      </a>
      {children}
    </article>
  );
}
