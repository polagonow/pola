// Client hydration for scenario 3. Plain React has no partial hydration, so the
// whole document root is hydrated; the interactive island is the Counter. The
// performance marks let a browser-side harness (Playwright/CDP) read hydration
// duration later; they are a no-op cost otherwise.

import React from "react";
import { hydrateRoot } from "react-dom/client";
import { Document, IslandBody } from "./App.jsx";

performance.mark("pola-bench:hydrate-start");

hydrateRoot(
  document,
  <Document title="Scenario 3 — Interactive Island">
    <IslandBody />
  </Document>,
);

requestAnimationFrame(() => {
  performance.mark("pola-bench:hydrate-end");
  try {
    performance.measure(
      "pola-bench:hydration",
      "pola-bench:hydrate-start",
      "pola-bench:hydrate-end",
    );
  } catch {
    /* marks may be absent under some conditions */
  }
});
