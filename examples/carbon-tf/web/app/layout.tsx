import React from "react";
import "./globals.scss";

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <title>carbon-tf</title>
      </head>
      <body>
        {children}
      </body>
    </html>
  );
}
