import React from "react";

type LayoutProps = {
  children: React.ReactNode
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div>
      app/pages/products/layout.tsx
      {children}
    </div>
  );
}
