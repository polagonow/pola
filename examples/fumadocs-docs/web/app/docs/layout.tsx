import type { ReactNode } from "react";
import { RootProvider } from "fumadocs-ui/provider/base";
import { PolaFumadocsProvider } from "@pola/fumadocs/provider";
import { DocsLayout } from "fumadocs-ui/layouts/docs";
import { Server } from "@pola/actions";
import { source } from "@/lib/source";

// Fumadocs context is scoped to the docs section:
//  - PolaFumadocsProvider maps fumadocs' routing/link onto Pola's client runtime
//  - RootProvider adds fumadocs-ui's theme/search context
//  - DocsLayout is the sidebar/nav shell, driven by the loader's page tree.
export default async function DocsRootLayout({ children }: { children: ReactNode }) {
  const { version } = await Server.getPolaVersion();
  const title = (
    <>
      Pola Docs <span style={{ color: "var(--color-muted)", fontWeight: 400, fontSize: "0.8em" }}>{version}</span>
    </>
  );
  return (
    <PolaFumadocsProvider>
      <RootProvider>
        <DocsLayout tree={source.pageTree} nav={{ title }}>
          {children}
        </DocsLayout>
      </RootProvider>
    </PolaFumadocsProvider>
  );
}
