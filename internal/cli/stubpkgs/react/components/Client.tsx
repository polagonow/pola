import { createRoot, hydrateRoot } from "react-dom/client";
import { createFromFetch } from "react-server-dom-esm/client";

function renderError(root: ReturnType<typeof createRoot>, msg: string) {
  root.render(<div className="rsc-err">{msg}</div>);
}

// Mount RSC tree into #root
const container = document.getElementById("root") ?? document.body;

try {
  const ssrData: string | undefined = (
    self as typeof globalThis & { __POLA_SSR_DATA__?: string }
  ).__POLA_SSR_DATA__;
  const fetchPromise = ssrData
    ? Promise.resolve(
        new Response(ssrData, {
          headers: { "Content-Type": "text/x-component" },
        }),
      )
    : fetch(location.pathname + location.search, {
        method: "GET",
        headers: { "Content-Type": "text/x-component" },
      });

  const hasServerHTML = container.childNodes.length > 0;

  createFromFetch(fetchPromise).then((comp: React.ReactNode) => {
    if (hasServerHTML) {
      // Selective hydration: React 18 will hydrate Suspense boundaries
      // independently and prioritize user-interacted components.
      hydrateRoot(container, comp);
    } else {
      // No server HTML — full client render (fallback path).
      const root = createRoot(container);
      root.render(comp);
    }
  });
} catch (err: unknown) {
  const root = createRoot(container);
  renderError(root, err instanceof Error ? err.message : String(err));
}
