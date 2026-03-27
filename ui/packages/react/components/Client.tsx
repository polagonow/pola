import { createRoot } from "react-dom/client";
import { createFromFetch } from "react-server-dom-esm/client";

// ── Server Action support ──────────────────────────────────────────────────
// callServer is the callback used by createServerReference stubs.
// It JSON-encodes arguments, POSTs them with the action ID header,
// and decodes the Flight response.
async function callServer(actionId: string, args: any[]): Promise<any> {
  const body = JSON.stringify(args);
  const response = fetch(location.pathname + location.search, {
    method: "POST",
    headers: {
      Accept: "text/x-component",
      "Content-Type": "application/json",
      "Next-Action": actionId,
    },
    body,
  });
  return createFromFetch(response, { callServer });
}

// Expose callServer globally so bundler-generated createServerReference
// stubs (in "use server" files) can reference it.
(globalThis as any).__callServer__ = callServer;

// ── Hydration ──────────────────────────────────────────────────────────────
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

  createFromFetch(fetchPromise, { callServer }).then((comp: React.ReactNode) => {
    root.render(comp);
  });
} catch (err: unknown) {
  renderError(root, err instanceof Error ? err.message : String(err));
}
