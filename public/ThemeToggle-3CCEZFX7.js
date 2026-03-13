"use client";
import {
  require_jsx_runtime
} from "./chunks/chunk-R6MJF37R.js";
import {
  __toESM,
  require_react
} from "./chunks/chunk-NQLUC6DA.js";

// app/components/ThemeToggle.tsx
var import_react = __toESM(require_react());
var import_jsx_runtime = __toESM(require_jsx_runtime());
function ThemeToggle() {
  const [dark, setDark] = (0, import_react.useState)(false);
  (0, import_react.useEffect)(() => {
    document.body.style.background = dark ? "#111" : "#f8f8f8";
    document.body.style.color = dark ? "#f0f0f0" : "#111";
  }, [dark]);
  return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(
    "button",
    {
      onClick: () => setDark((d) => !d),
      style: {
        padding: "0.4rem 1rem",
        borderRadius: "6px",
        border: "1px solid #ccc",
        cursor: "pointer",
        background: dark ? "#333" : "#fff",
        color: dark ? "#fff" : "#111",
        fontWeight: 500
      },
      children: dark ? "\u2600\uFE0F Light" : "\u{1F319} Dark"
    }
  );
}
export {
  ThemeToggle as default
};
