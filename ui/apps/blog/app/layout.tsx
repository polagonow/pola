import React from "react";
import Nav from "@/components/Nav";
import ThemeToggle from "@/components/ThemeToggle";

// Root layout — wraps every page with the full shell: topnav + main content area.
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="shell">
      <header className="topnav">
        <a href="/" className="brand">DevBlog</a>
        <Nav />
        <div className="actions"><ThemeToggle /></div>
      </header>
      <div className="main">{children}</div>
    </div>
  );
}
