// Browser entry for the RSC baseline: fetch the Flight stream and render it into
// #root with createRoot — the same client model Pola uses (createFromFetch +
// client render, not hydration of server HTML). Minimal webpack shims are
// provided because these scenarios ship no client components.

import React, { use } from "react";
import { createRoot } from "react-dom/client";
import { createFromFetch } from "react-server-dom-webpack/client.browser";

// No "use client" modules in scenarios 1/2/4, so the client never loads a chunk.
globalThis.__webpack_require__ = (id) => ({ default: () => null, [id]: () => null });
globalThis.__webpack_chunk_load__ = () => Promise.resolve();

performance.mark("pola-bench:rsc-fetch-start");
const treePromise = createFromFetch(
  fetch(location.pathname + location.search, {
    headers: { Accept: "text/x-component" },
  }),
  { callServer() { throw new Error("server actions not used in the RSC baseline"); } },
);

function Shell() {
  return use(treePromise);
}

const root = createRoot(document.getElementById("root"));
root.render(<Shell />);

treePromise.then(() => {
  performance.mark("pola-bench:rsc-interactive");
  try {
    performance.measure("pola-bench:rsc", "pola-bench:rsc-fetch-start", "pola-bench:rsc-interactive");
  } catch {
    /* marks may be absent */
  }
});
