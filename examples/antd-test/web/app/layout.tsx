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
        <title>antd-test</title>
      </head>
      <body>
        {children}
      </body>
    </html>
  );
}
