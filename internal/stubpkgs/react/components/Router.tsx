"use client";

// Next.js-style client router hooks. Vendored and adapted from rari (rari/router):
// rari reaches its runtime's navigate via "rari:register-navigate" events, whereas
// Pola exposes window.__pola_navigate__ (set by @pola/react/client) directly. The
// provider stays in sync by listening for the "pola:navigate" event the client
// runtime dispatches on every SPA navigation (Link clicks, programmatic, back/forward).

import { createContext, use, useEffect, useMemo, useState, type ReactNode } from "react";

export interface NavigationOptions {
  replace?: boolean;
  scroll?: boolean;
}

export interface RouterContextValue {
  pathname: string;
  searchParams: URLSearchParams;
  params: Record<string, string | string[]>;
  push: (href: string, options?: NavigationOptions) => Promise<void>;
  replace: (href: string, options?: NavigationOptions) => Promise<void>;
  back: () => void;
  forward: () => void;
  refresh: () => void;
  prefetch: (href: string) => Promise<void>;
}

const RouterContext = createContext<RouterContextValue | null>(null);

export interface RouterProviderProps {
  children: ReactNode;
  initialPathname?: string;
}

function polaNavigate(): ((href: string) => void) | undefined {
  return (self as typeof globalThis & { __pola_navigate__?: (href: string) => void }).__pola_navigate__;
}

/**
 * navigate performs a client-side navigation through Pola's Flight SPA runtime,
 * falling back to a full page load when the runtime isn't present.
 */
export async function navigate(href: string, options?: NavigationOptions): Promise<void> {
  if (typeof window === "undefined") return;
  const nav = polaNavigate();
  if (!nav) {
    if (options?.replace) window.location.replace(href);
    else window.location.href = href;
    return;
  }
  if (options?.replace) window.history.replaceState(null, "", href);
  nav(href);
}

export function RouterProvider({ children, initialPathname }: RouterProviderProps) {
  const [pathname, setPathname] = useState(() =>
    typeof window !== "undefined" ? window.location.pathname : (initialPathname ?? "/"),
  );
  const [searchParams, setSearchParams] = useState(() =>
    typeof window !== "undefined" ? new URLSearchParams(window.location.search) : new URLSearchParams(),
  );

  useEffect(() => {
    const sync = () => {
      setPathname(window.location.pathname);
      setSearchParams(new URLSearchParams(window.location.search));
    };
    window.addEventListener("pola:navigate", sync);
    window.addEventListener("popstate", sync);
    return () => {
      window.removeEventListener("pola:navigate", sync);
      window.removeEventListener("popstate", sync);
    };
  }, []);

  const value = useMemo<RouterContextValue>(
    () => ({
      pathname,
      searchParams,
      // Route params are provided to pages server-side via props; the client
      // router does not recompute them (matching rari's client behavior).
      params: {},
      push: (href, options) => navigate(href, options),
      replace: (href, options) => navigate(href, { ...options, replace: true }),
      back: () => window.history.back(),
      forward: () => window.history.forward(),
      refresh: () => {
        const nav = polaNavigate();
        if (nav) nav(window.location.pathname + window.location.search);
      },
      prefetch: async (href: string) => {
        try {
          const url = new URL(href, window.location.origin);
          await fetch(url.pathname + url.search, {
            headers: { Accept: "text/x-component" },
            priority: "low",
          } as RequestInit);
        } catch {
          /* prefetch is best-effort */
        }
      },
    }),
    [pathname, searchParams],
  );

  return <RouterContext value={value}>{children}</RouterContext>;
}

export function useRouter(): RouterContextValue {
  const context = use(RouterContext);
  if (!context) throw new Error("useRouter must be used within a RouterProvider");
  return context;
}

export function usePathname(): string {
  return useRouter().pathname;
}

export function useSearchParams(): URLSearchParams {
  return useRouter().searchParams;
}

export function useParams(): Record<string, string | string[]> {
  return useRouter().params;
}
