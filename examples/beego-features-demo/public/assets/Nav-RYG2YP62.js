"use client";
import {
  __toESM,
  require_jsx_runtime
} from "./chunks/chunk-6RVRE7CD.js";

// components/Nav.tsx
var import_jsx_runtime = __toESM(require_jsx_runtime());
var LINKS = [
  { href: "/", label: "Home" },
  { href: "/login", label: "Login" },
  { href: "/profile", label: "Profile" }
];
function Nav() {
  return /* @__PURE__ */ (0, import_jsx_runtime.jsx)("nav", { className: "flex items-center gap-4", children: LINKS.map(({ href, label }) => /* @__PURE__ */ (0, import_jsx_runtime.jsx)("a", { href, className: "text-[var(--color-muted)] text-sm font-medium hover:text-[var(--color-fg)] hover:no-underline", children: label }, href)) });
}
export {
  Nav as default
};
