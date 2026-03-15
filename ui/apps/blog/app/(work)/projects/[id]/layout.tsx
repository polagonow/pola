import React from "react";

export default function ProjectLayout({ children }: { children: React.ReactNode }) {
  return (
    <div>
      <a href="/projects" className="back-link">← All projects</a>
      {children}
    </div>
  );
}
