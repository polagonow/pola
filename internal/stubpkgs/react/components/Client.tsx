import { createRoot } from "react-dom/client";
import { createFromFetch } from "./flight";
import React, { use, startTransition } from "react";
import { notifyPathnameChange } from "./Link";

// ── Flight data fetching ────────────────────────────────────────────────────

const FLIGHT_CONTENT_TYPE = "text/x-component";

// callServer invokes a server action referenced from the Flight stream (a
// 'use server' function passed as a prop). The reference id is the registry key
// "moduleId:exportName"; split it and POST to /_pola/action.
async function callServer(id: string, args: unknown[]): Promise<unknown> {
  const sep = id.lastIndexOf(":");
  const moduleId = sep >= 0 ? id.slice(0, sep) : id;
  const exportName = sep >= 0 ? id.slice(sep + 1) : id;
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (typeof document !== "undefined") {
    const csrf = document.querySelector('meta[name="csrf-token"]')?.getAttribute("content");
    if (csrf) headers["X-CSRF-Token"] = csrf;
  }
  const resp = await fetch("/_pola/action", {
    method: "POST",
    headers,
    body: JSON.stringify({ id: moduleId, export_name: exportName, args }),
  });
  const data = await resp.json();
  if (data.redirect) {
    window.location.assign(data.redirect);
    return { redirect: data.redirect };
  }
  if (!data.success) throw new Error(data.error || `Server action "${exportName}" failed`);
  return data.result;
}

const flightOptions = { callServer };

function fetchFlight(url: string, signal?: AbortSignal): Promise<React.ReactNode> {
  const resp = fetch(url, {
    method: "GET",
    // Accept is the reliable signal (browsers omit Content-Type on bodyless
    // GETs); the server treats either header as an RSC/flight request.
    headers: { Accept: FLIGHT_CONTENT_TYPE, "Content-Type": FLIGHT_CONTENT_TYPE },
    signal,
    cache: "no-store",
  });
  return createFromFetch(resp, flightOptions);
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
    return createFromFetch(resp, flightOptions);
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
  // Notify @pola/react/router consumers (useRouter/usePathname) of the change.
  // Back/forward fire a native "popstate" the router also listens to.
  self.dispatchEvent(new CustomEvent("pola:navigate"));
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
