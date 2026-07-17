"use client";

import { CircleIcon } from "lucide-react";
import { UserMenu } from "@/components/user-menu";

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <section className="flex flex-col min-h-screen">
      <header className="border-b border-gray-200 bg-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex justify-between items-center">
          <a href="/" className="flex items-center">
            <CircleIcon className="h-6 w-6 text-orange-500" />
            <span className="ml-2 text-xl font-semibold text-gray-900">
              ACME
            </span>
          </a>
          <div className="flex items-center space-x-4">
            <UserMenu />
          </div>
        </div>
      </header>
      {children}
    </section>
  );
}
