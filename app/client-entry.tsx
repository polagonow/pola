import { createRoot } from "react-dom/client";
import { createFromFetch } from "react-server-dom-esm/client";

const container = document.getElementById("root");

if (!container) {
  throw new Error("Root element not found");
}

const root = createRoot(container);

createFromFetch(
  fetch(location.pathname + location.search, {
    method: "GET",
    headers: {
      "Content-Type": "text/x-component",
    },
  })
  // @ts-expect-error type
).then((comp) => {
  root.render(comp);
});
