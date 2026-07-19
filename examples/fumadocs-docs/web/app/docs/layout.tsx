import type { ReactNode } from "react";
import { RootProvider } from "fumadocs-ui/provider/base";
import { PolaFumadocsProvider } from "@pola/fumadocs/provider";
import { DocsLayout } from "fumadocs-ui/layouts/docs";
import { source } from "@/lib/source";

// Fumadocs context is scoped to the docs section:
//  - PolaFumadocsProvider maps fumadocs' routing/link onto Pola's client runtime
//  - RootProvider adds fumadocs-ui's theme/search context
//  - DocsLayout is the sidebar/nav shell, driven by the loader's page tree.
export default function DocsRootLayout({ children }: { children: ReactNode }) {
  return (
    <PolaFumadocsProvider>
      <RootProvider>
        <DocsLayout tree={source.pageTree} nav={{ title: "Pola Docs" }}>
          {children}
        </DocsLayout>
      </RootProvider>
    </PolaFumadocsProvider>
  );
}
