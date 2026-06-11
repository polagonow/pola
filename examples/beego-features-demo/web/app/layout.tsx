import React from "react";
import "./globals.css";
import Nav from "@/components/Nav";
import FlashMessage from "@/components/FlashMessage";

export const metadata = {
  title: { default: "Beego Features Demo", template: "%s | Demo" },
  description: "Demonstrating session, rate limiting, validation, i18n, flash messages, and more.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-[var(--color-border)] px-6 py-3 flex items-center gap-8">
        <a href="/" className="font-bold text-lg text-[var(--color-fg)] hover:no-underline">
          Pola Features
        </a>
        <Nav />
      </header>
      <div className="flex-1 max-w-[800px] mx-auto w-full px-6 py-8">
        <FlashMessage />
        {children}
      </div>
    </div>
  );
}
