"use client";
import {
  Auth
} from "./chunks/chunk-DK7YOZM5.js";
import {
  __toESM,
  require_jsx_runtime,
  require_react
} from "./chunks/chunk-6RVRE7CD.js";

// components/LogoutButton.tsx
var import_react = __toESM(require_react());
var import_jsx_runtime = __toESM(require_jsx_runtime());
function LogoutButton() {
  const [isPending, startTransition] = (0, import_react.useTransition)();
  function handleLogout() {
    startTransition(async () => {
      await Auth.logout();
      window.location.href = "/";
    });
  }
  return /* @__PURE__ */ (0, import_jsx_runtime.jsx)("button", { onClick: handleLogout, disabled: isPending, className: "px-4 py-2 border border-[var(--color-border)] rounded-lg text-sm font-medium cursor-pointer bg-transparent text-[var(--color-fg)] disabled:opacity-60", children: isPending ? "Logging out..." : "Logout" });
}
export {
  LogoutButton as default
};
