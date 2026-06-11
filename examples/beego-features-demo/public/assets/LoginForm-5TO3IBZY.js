"use client";
import {
  external_exports,
  u,
  useForm
} from "./chunks/chunk-7D3LSKAS.js";
import {
  csrfToken
} from "./chunks/chunk-ZXZXPQXQ.js";
import {
  __toESM,
  require_jsx_runtime,
  require_react
} from "./chunks/chunk-6RVRE7CD.js";

// components/LoginForm.tsx
var import_react = __toESM(require_react());

// schemas/login.ts
var loginSchema = external_exports.object({
  username: external_exports.string().min(1, "Username is required"),
  password: external_exports.string().min(1, "Password is required")
});

// components/LoginForm.tsx
var import_jsx_runtime = __toESM(require_jsx_runtime());
function LoginForm() {
  const [error, setError] = (0, import_react.useState)(null);
  const [isPending, startTransition] = (0, import_react.useTransition)();
  const { register, handleSubmit, formState: { errors } } = useForm({
    resolver: u(loginSchema)
  });
  function onSubmit(data) {
    setError(null);
    startTransition(async () => {
      try {
        const res = await fetch("/login", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrfToken()
          },
          body: JSON.stringify(data)
        });
        if (!res.ok) {
          const body = await res.json();
          throw new Error(body.error || res.statusText);
        }
        window.location.href = "/profile";
      } catch (err) {
        setError(err instanceof Error ? err.message : "Login failed");
      }
    });
  }
  return /* @__PURE__ */ (0, import_jsx_runtime.jsxs)(import_jsx_runtime.Fragment, { children: [
    error && /* @__PURE__ */ (0, import_jsx_runtime.jsx)("div", { className: "px-4 py-3 mb-4 rounded-lg text-sm", style: { background: "#fef2f2", border: "1px solid #fecaca", color: "#dc2626" }, children: error }),
    /* @__PURE__ */ (0, import_jsx_runtime.jsxs)("form", { onSubmit: handleSubmit(onSubmit), className: "flex flex-col gap-4", children: [
      /* @__PURE__ */ (0, import_jsx_runtime.jsxs)("div", { children: [
        /* @__PURE__ */ (0, import_jsx_runtime.jsx)("label", { htmlFor: "username", className: "block text-sm font-medium mb-1", children: "Username" }),
        /* @__PURE__ */ (0, import_jsx_runtime.jsx)("input", { type: "text", id: "username", ...register("username"), className: "w-full px-3 py-2 border border-[var(--color-border)] rounded-lg text-sm bg-[var(--color-bg)]" }),
        errors.username && /* @__PURE__ */ (0, import_jsx_runtime.jsx)("p", { className: "text-xs mt-1", style: { color: "#dc2626" }, children: errors.username.message })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime.jsxs)("div", { children: [
        /* @__PURE__ */ (0, import_jsx_runtime.jsx)("label", { htmlFor: "password", className: "block text-sm font-medium mb-1", children: "Password" }),
        /* @__PURE__ */ (0, import_jsx_runtime.jsx)("input", { type: "password", id: "password", ...register("password"), className: "w-full px-3 py-2 border border-[var(--color-border)] rounded-lg text-sm bg-[var(--color-bg)]" }),
        errors.password && /* @__PURE__ */ (0, import_jsx_runtime.jsx)("p", { className: "text-xs mt-1", style: { color: "#dc2626" }, children: errors.password.message })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime.jsx)("button", { type: "submit", disabled: isPending, className: "px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-sm font-medium cursor-pointer border-none disabled:opacity-60", children: isPending ? "Logging in..." : "Login" })
    ] })
  ] });
}
export {
  LoginForm as default
};
