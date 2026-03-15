import React from "react";

// Route group layout: (marketing)
// This layout wraps only routes inside (marketing)/ — it does NOT affect
// routes outside the group. The "(marketing)" folder name is invisible in
// the URL; /docs is still served at /docs, not /(marketing)/docs.
export default function MarketingLayout({ children }: { children: React.ReactNode }) {
  return (
    <div>
      app/(marketing)/layout.tsx
      <div className="banner">
        <span>📣 Marketing Section</span>
      </div>
      {children}
    </div>
  );
}
