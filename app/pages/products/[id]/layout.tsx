import React from "react";

type LayoutProps = {
  children: React.ReactNode
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div>
      app/pages/products/[id]/layout.tsx
      {children}
    </div>
  );
}
