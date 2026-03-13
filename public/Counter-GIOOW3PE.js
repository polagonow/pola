"use client";
import {
  require_jsx_runtime
} from "./chunks/chunk-Y6XQMWWR.js";
import {
  __toESM,
  require_react
} from "./chunks/chunk-MTWJ6HQK.js";

// app/components/Counter.tsx
var import_react = __toESM(require_react());
var import_jsx_runtime = __toESM(require_jsx_runtime());
function Counter({ initialCount = 0, label = "Count" }) {
  const [count, setCount] = (0, import_react.useState)(initialCount);
  return /* @__PURE__ */ (0, import_jsx_runtime.jsxs)("div", { style: {
    display: "inline-flex",
    alignItems: "center",
    gap: "0.75rem",
    background: "#fff",
    border: "1px solid #e0e0e0",
    borderRadius: "8px",
    padding: "0.5rem 1rem"
  }, children: [
    /* @__PURE__ */ (0, import_jsx_runtime.jsxs)("span", { style: { fontWeight: 600, minWidth: "4rem" }, children: [
      label,
      ": ",
      count
    ] }),
    /* @__PURE__ */ (0, import_jsx_runtime.jsx)(
      "button",
      {
        onClick: () => setCount((c) => c - 1),
        style: { padding: "0.25rem 0.75rem", borderRadius: "4px", border: "1px solid #ccc", cursor: "pointer" },
        children: "\u2212"
      }
    ),
    /* @__PURE__ */ (0, import_jsx_runtime.jsx)(
      "button",
      {
        onClick: () => setCount((c) => c + 1),
        style: { padding: "0.25rem 0.75rem", borderRadius: "4px", border: "1px solid #ccc", cursor: "pointer", background: "#0070f3", color: "#fff", borderColor: "#0070f3" },
        children: "+"
      }
    )
  ] });
}
export {
  Counter as default
};
