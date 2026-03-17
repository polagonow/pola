import React from "react";

export default function PostLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <article>
      <a href="/posts" className="back-link">
        ← All posts
      </a>
      {children}
    </article>
  );
}
