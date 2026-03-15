import { createRoot } from "react-dom/client";
import { createFromFetch } from "react-server-dom-esm/client";
import ThemeToggle from "@/components/ThemeToggle";
import Nav from "@/components/Nav";

// Mount RSC tree into #root
const container = document.getElementById("root");
if (!container) throw new Error("Root element not found");
const root = createRoot(container);

const flightData: string | undefined = (self as any).__flight_data;
const fetchPromise = flightData
  ? Promise.resolve(new Response(flightData, { headers: { "Content-Type": "text/x-component" } }))
  : fetch(location.pathname + location.search, { method: "GET", headers: { "Content-Type": "text/x-component" } });

createFromFetch(fetchPromise).then((comp: React.ReactNode) => {
  root.render(comp);
});

// Mount Nav into the nav slot
const navSlot = document.getElementById("nav-root");
if (navSlot) {
  createRoot(navSlot).render(<Nav />);
}

// Mount ThemeToggle into the nav action slot
const toggleSlot = document.getElementById("theme-toggle-root");
if (toggleSlot) {
  createRoot(toggleSlot).render(<ThemeToggle />);
}
