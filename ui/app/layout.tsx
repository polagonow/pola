import React from "react";

type LayoutProps = {
  children: React.ReactNode
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div>
      app/layout.tsx
      {children}
    </div>
  );
}
