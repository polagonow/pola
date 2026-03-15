import React from "react";

export default function AboutLayout({ children }: { children: React.ReactNode }) {
  return (
    <div>
      <div className="section-badge">ℹ About</div>
      {children}
    </div>
  );
}
