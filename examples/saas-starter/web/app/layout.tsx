import React from "react";
import "./globals.css";

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <title>ACME — SaaS Starter</title>
      </head>
      <body className="min-h-[100dvh] bg-gray-50">{children}</body>
    </html>
  );
}
