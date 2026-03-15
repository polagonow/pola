import React from "react";

type LayoutProps = {
  children: React.ReactNode
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div>
      app/products/[id]/layout.tsx
      {children}
    </div>
  );
}
