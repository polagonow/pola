import { createRoot } from "react-dom/client";
import { createFromFetch } from "react-server-dom-esm/client";

function renderError(root: ReturnType<typeof createRoot>, msg: string) {
  root.render(<div className="rsc-err">{msg}</div>);
}

// Mount RSC tree into #root
const container = document.getElementById("root") ?? document.body;
const root = createRoot(container);

try {
  const flightData: string | undefined = (
    self as typeof globalThis & { __flight_data?: string }
  ).__flight_data;
  const fetchPromise = flightData
    ? Promise.resolve(
        new Response(flightData, {
          headers: { "Content-Type": "text/x-component" },
        }),
      )
    : fetch(location.pathname + location.search, {
        method: "GET",
        headers: { "Content-Type": "text/x-component" },
      });

  createFromFetch(fetchPromise).then((comp: React.ReactNode) => {
    root.render(comp);
  });
} catch (err: unknown) {
  renderError(root, err instanceof Error ? err.message : String(err));
}
