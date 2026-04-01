"use client";

import React, { useCallback, useEffect, useRef, useSyncExternalStore, type AnchorHTMLAttributes, type ReactNode } from "react";

type ClassFn = (state: { isActive: boolean; isPending: boolean }) => string;

interface LinkProps extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "href" | "className"> {
  href: string;
  children?: ReactNode;
  className?: string | ClassFn;
  /** When true, prefetch the Flight data on hover/viewport so navigation is instant. */
  prefetch?: boolean;
}

/**
 * Client-side navigation link that fetches RSC Flight data without a full
 * page reload. Falls back to native <a> behaviour for modifier-clicks and
 * non-local hrefs.
 *
 * Supports active-link detection: pass a function as className to receive
 * { isActive, isPending } and return the appropriate class string.
 */
// Track pathname reactively so Link re-renders after SPA navigations.
const listeners = new Set<() => void>();
const subscribe = (cb: () => void) => {
  listeners.add(cb);
  return () => listeners.delete(cb);
};
const getPathname = () =>
  typeof window !== "undefined" ? window.location.pathname : "";

// Notify all Link instances when the URL changes.
export function notifyPathnameChange() {
  for (const cb of listeners) cb();
}

// Hook into popstate so back/forward also updates active state.
if (typeof window !== "undefined") {
  window.addEventListener("popstate", () => notifyPathnameChange());
}

// Prefetch cache — avoids duplicate fetches for the same href.
const prefetched = new Set<string>();

function prefetchFlight(href: string) {
  if (prefetched.has(href)) return;
  try {
    const url = new URL(href, location.origin);
    if (url.origin !== location.origin) return;
  } catch {
    return;
  }
  prefetched.add(href);
  // Low-priority fetch; result is cached by the browser for the subsequent navigation fetch.
  fetch(href, {
    method: "GET",
    headers: { "Content-Type": "text/x-component" },
    priority: "low",
  } as RequestInit).catch(() => {
    // Remove from set so it can be retried.
    prefetched.delete(href);
  });
}

export function Link({ href, children, className, onClick, prefetch, ...props }: LinkProps) {
  const pathname = useSyncExternalStore(subscribe, getPathname, () => "");
  const isActive = !href
    ? false
    : href === "/"
      ? pathname === "/"
      : pathname === href || pathname.startsWith(href + "/");
  const isPending = false;

  const resolved =
    typeof className === "function"
      ? className({ isActive, isPending })
      : (className ?? "");

  const ref = useRef<HTMLAnchorElement>(null);

  // Prefetch on viewport intersection when prefetch={true}.
  useEffect(() => {
    if (!prefetch || !ref.current) return;
    const el = ref.current;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          prefetchFlight(href);
          observer.disconnect();
        }
      },
      { rootMargin: "200px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [prefetch, href]);

  const handleMouseEnter = useCallback(() => {
    if (prefetch) prefetchFlight(href);
  }, [prefetch, href]);

  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLAnchorElement>) => {
      onClick?.(e);
      if (e.defaultPrevented) return;

      // Let the browser handle modifier-clicks and non-left-clicks.
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0) return;

      // Only intercept same-origin, non-hash navigations.
      try {
        const url = new URL(href, location.origin);
        if (url.origin !== location.origin) return;
      } catch {
        return;
      }

      e.preventDefault();

      const nav = (self as typeof globalThis & { __pola_navigate__?: (href: string) => void }).__pola_navigate__;
      if (nav) {
        nav(href);
      } else {
        location.href = href;
      }
    },
    [href, onClick],
  );

  return (
    <a ref={ref} href={href} className={resolved} onClick={handleClick} onMouseEnter={handleMouseEnter} {...props}>
      {children}
    </a>
  );
}

export default Link;
