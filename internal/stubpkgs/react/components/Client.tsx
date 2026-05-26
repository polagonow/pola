import { createRoot, hydrateRoot } from "react-dom/client";
import { createFromFetch } from "react-server-dom-esm/client";
import React, { use, startTransition } from "react";
import { notifyPathnameChange } from "./Link";

// ── Flight data fetching ────────────────────────────────────────────────────

const FLIGHT_CONTENT_TYPE = "text/x-component";

function fetchFlight(url: string, signal?: AbortSignal): Promise<React.ReactNode> {
  const resp = fetch(url, {
    method: "GET",
    headers: { "Content-Type": FLIGHT_CONTENT_TYPE },
    signal,
    cache: "no-store",
  });
  return createFromFetch(resp);
}

function initialFlight(): Promise<React.ReactNode> {
  const ssrData: string | undefined = (
    self as typeof globalThis & { __POLA_SSR_DATA__?: string }
  ).__POLA_SSR_DATA__;
  if (ssrData) {
    const resp = Promise.resolve(
      new Response(ssrData, {
        headers: { "Content-Type": FLIGHT_CONTENT_TYPE },
      }),
    );
    return createFromFetch(resp);
  }
  return fetchFlight(location.pathname + location.search);
}

// ── Navigation manager ──────────────────────────────────────────────────────

let treePromise = initialFlight();
let navigationController: AbortController | null = null;

function navigateWithFlight(url: string) {
  // Abort any in-flight navigation fetch.
  navigationController?.abort();
  const controller = new AbortController();
  navigationController = controller;

  startTransition(() => {
    treePromise = fetchFlight(url, controller.signal);
    root.render(<Shell />);
  });
}

function navigate(href: string) {
  history.pushState(null, "", href);
  notifyPathnameChange();
  navigateWithFlight(href);
}

// Expose navigate globally so <Link> can call it.
(self as typeof globalThis & { __pola_navigate__?: typeof navigate }).__pola_navigate__ = navigate;

// Handle browser back/forward.
window.addEventListener("popstate", () => {
  notifyPathnameChange();
  navigateWithFlight(location.pathname + location.search);
});

// ── Shell component ─────────────────────────────────────────────────────────

function Shell() {
  return use(treePromise);
}

// ── Mount ───────────────────────────────────────────────────────────────────

const container = document.getElementById("__POLA_ROOT__") ?? document.body;
const root = createRoot(container);
root.render(<Shell />);
