import React from "react";

// Route group layout: (dashboard)
// Shared layout for all dashboard routes (/settings, /profile, etc.)
// without adding a URL segment.
export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="dashboard">
      app/(dashboard)/layout.tsx
      <nav className="sidebar">
        <a href="/settings">Settings</a>
      </nav>
      <main>{children}</main>
    </div>
  );
}
