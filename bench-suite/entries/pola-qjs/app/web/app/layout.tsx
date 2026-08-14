import React from "react";
import "./globals.css";

// Minimal root layout — kept close to the control's document so the conformance
// gate compares content, not chrome.
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <title>Pola benchmark</title>
      </head>
      <body>{children}</body>
    </html>
  );
}
