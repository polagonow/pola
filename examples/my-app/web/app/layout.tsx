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
        <title>my-app</title>
      </head>
      <body>
        <div className="flex min-h-screen flex-col">
          <header className="flex items-center gap-8 border-b px-6 py-3">
            <a href="/" className="text-lg font-bold no-underline hover:no-underline">
              my-app
            </a>
          </header>
          <div className="mx-auto flex-1 w-full max-w-[1000px] px-6 py-8">
            {children}
          </div>
        </div>
      </body>
    </html>
  );
}
