"use client";

// Pola adapter for `fumadocs-core/framework`. It supplies the routing/link
// primitives fumadocs-ui needs (usePathname / useRouter / useParams / Link),
// mapped onto Pola's client runtime — `window.__pola_navigate__` for SPA
// navigation and the `pola:navigate` / `popstate` events for pathname changes.
//
// It is deliberately self-contained: Pola apps do not mount @pola/react's
// RouterProvider, so this reads `window.location` directly rather than depending
// on that context. Wrap your app in <PolaFumadocsProvider> (typically in the
// root layout, outside fumadocs-ui's RootProvider).

import {
  useMemo,
  useSyncExternalStore,
  type ComponentProps,
  type ReactNode,
} from "react";
import { FrameworkProvider } from "fumadocs-core/framework";
import { Link as PolaLink } from "@pola/react/link";
import { navigate } from "@pola/react/router";

function subscribe(onChange: () => void): () => void {
  if (typeof window === "undefined") return () => {};
  window.addEventListener("pola:navigate", onChange);
  window.addEventListener("popstate", onChange);
  return () => {
    window.removeEventListener("pola:navigate", onChange);
    window.removeEventListener("popstate", onChange);
  };
}

function currentPathname(): string {
  return typeof window !== "undefined" ? window.location.pathname : "/";
}

const framework = {
  usePathname(): string {
    return useSyncExternalStore(subscribe, currentPathname, () => "/");
  },
  useParams(): Record<string, string | string[]> {
    // Route params reach pages server-side via props; the client router does
    // not recompute them (matches @pola/react's behaviour).
    return {};
  },
  useRouter() {
    return useMemo(
      () => ({
        push(url: string) {
          void navigate(url);
        },
        refresh() {
          void navigate(currentPathname());
        },
      }),
      [],
    );
  },
  Link({
    href,
    prefetch,
    ...props
  }: ComponentProps<"a"> & { prefetch?: boolean }) {
    return <PolaLink href={href ?? "#"} prefetch={prefetch} {...props} />;
  },
};

export function PolaFumadocsProvider({ children }: { children: ReactNode }) {
  return <FrameworkProvider {...framework}>{children}</FrameworkProvider>;
}

export default PolaFumadocsProvider;
