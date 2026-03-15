import React from "react";

type LayoutProps = {
  children: React.ReactNode
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div>
      app/pages/user/layout.tsx
      {children}
    </div>
  );
}
