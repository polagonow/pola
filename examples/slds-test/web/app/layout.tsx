import React from "react";
import "@salesforce-ux/design-system/assets/styles/salesforce-lightning-design-system.min.css";
import "./globals.css";
import SLDSProvider from "@/lib/slds-provider";

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <title>slds-test</title>
      </head>
      <body>
        <SLDSProvider>
          <div className="slds-grid slds-grid_vertical" style={{ minHeight: "100vh" }}>
            <header className="slds-global-header_container">
              <div className="slds-global-header slds-grid slds-grid_align-spread slds-grid_vertical-align-center">
                <div className="slds-global-header__item">
                  <span className="slds-text-heading_small">slds-test</span>
                </div>
              </div>
            </header>
            <main className="slds-p-around_large" style={{ flex: 1 }}>
              {children}
            </main>
          </div>
        </SLDSProvider>
      </body>
    </html>
  );
}
