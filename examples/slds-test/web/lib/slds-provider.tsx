"use client";

import IconSettings from "@salesforce/design-system-react/components/icon-settings";

export default function SLDSProvider({ children }: { children: React.ReactNode }) {
  return (
    <IconSettings iconPath="/assets/icons">
      {children}
    </IconSettings>
  );
}
