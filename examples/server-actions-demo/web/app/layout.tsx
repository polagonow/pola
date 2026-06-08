import React from "react";

// Root layout. The nativersc renderer wraps this in the HTML document shell, so
// the layout just provides the page chrome.
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        fontFamily: "system-ui, -apple-system, sans-serif",
        color: "#1a1a1a",
        background: "#fafafa",
        minHeight: "100vh",
      }}
    >
      {children}
    </div>
  );
}
