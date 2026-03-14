import { createRoot } from "react-dom/client";
import { createFromFetch } from "react-server-dom-esm/client";

const container = document.getElementById("root");

if (!container) {
  throw new Error("Root element not found");
}

const root = createRoot(container);

const flightData: string | undefined = (self as any).__flight_data;

const fetchPromise = flightData
  ? Promise.resolve(
      new Response(flightData, {
        headers: { "Content-Type": "text/x-component" },
      })
    )
  : fetch(location.pathname + location.search, {
      method: "GET",
      headers: { "Content-Type": "text/x-component" },
    });

// @ts-expect-error type error
createFromFetch(fetchPromise).then((comp) => {
  root.render(comp);
});
