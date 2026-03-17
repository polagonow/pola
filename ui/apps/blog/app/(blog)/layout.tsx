import React from "react";

// (blog) route group layout — sidebar with tag navigation for /posts and /posts/:slug
export default function BlogLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="sidebar-layout">
      <aside className="sidebar">
        <h4>Topics</h4>
        <ul>
          <li>
            <a href="/posts">All posts</a>
          </li>
          <li>
            <a href="/posts?tag=go">Go</a>
          </li>
          <li>
            <a href="/posts?tag=react">React</a>
          </li>
          <li>
            <a href="/posts?tag=rsc">RSC</a>
          </li>
          <li>
            <a href="/posts?tag=vm">VM</a>
          </li>
        </ul>
      </aside>
      <div>{children}</div>
    </div>
  );
}
