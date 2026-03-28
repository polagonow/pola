import React from "react";
import "./globals.css";

import Nav from "@/components/Nav";
import ThemeToggle from "@/components/ThemeToggle";

// Root layout — wraps every page with the full shell: topnav + main content area.
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-[var(--color-border)] px-6 py-3 flex items-center gap-8">
        <a href="/" className="font-bold text-lg text-[var(--color-fg)]">
          DevBlog
        </a>
        <Nav />
        <div className="ml-auto flex items-center gap-3">
          <ThemeToggle />
        </div>
      </header>
      <div className="flex-1 max-w-[1000px] mx-auto w-full px-6 py-8">{children}</div>
    </div>
  );
}
