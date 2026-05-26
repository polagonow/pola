import React from "react";

// (blog) route group layout — sidebar with tag navigation for /posts and /posts/:slug
export default function BlogLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="grid grid-cols-[200px_1fr] gap-8 items-start max-sm:grid-cols-1">
      <aside className="sticky top-6 max-sm:hidden">
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
