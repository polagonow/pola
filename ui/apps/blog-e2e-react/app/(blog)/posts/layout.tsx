import React from "react";

export default function PostsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="section-badge">✍︎ Blog</div>
      {children}
    </div>
  );
}
