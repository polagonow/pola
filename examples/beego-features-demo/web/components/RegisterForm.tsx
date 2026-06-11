"use client";

import { useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { registerSchema, type RegisterInput } from "@/schemas/register";
import { csrfToken } from "@/utils/csrf";

export default function RegisterForm() {
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const { register, handleSubmit, formState: { errors } } = useForm<RegisterInput>({
    resolver: zodResolver(registerSchema),
  });

  function onSubmit(data: RegisterInput) {
    setError(null);
    startTransition(async () => {
      try {
        const res = await fetch("/register", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrfToken(),
          },
          body: JSON.stringify(data),
        });
        if (!res.ok) {
          const body = await res.json();
          throw new Error(body.error || res.statusText);
        }
        window.location.href = "/profile";
      } catch (err) {
        setError(err instanceof Error ? err.message : "Registration failed");
      }
    });
  }

  return (
    <>
      {error && (
        <div className="px-4 py-3 mb-4 rounded-lg text-sm" style={{ background: "#fef2f2", border: "1px solid #fecaca", color: "#dc2626" }}>
          {error}
        </div>
      )}
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <div>
          <label htmlFor="username" className="block text-sm font-medium mb-1">Username</label>
          <input type="text" id="username" {...register("username")} className="w-full px-3 py-2 border border-[var(--color-border)] rounded-lg text-sm bg-[var(--color-bg)]" />
          {errors.username && <p className="text-xs mt-1" style={{ color: "#dc2626" }}>{errors.username.message}</p>}
        </div>
        <div>
          <label htmlFor="displayName" className="block text-sm font-medium mb-1">Display Name</label>
          <input type="text" id="displayName" {...register("displayName")} className="w-full px-3 py-2 border border-[var(--color-border)] rounded-lg text-sm bg-[var(--color-bg)]" />
          {errors.displayName && <p className="text-xs mt-1" style={{ color: "#dc2626" }}>{errors.displayName.message}</p>}
        </div>
        <div>
          <label htmlFor="password" className="block text-sm font-medium mb-1">Password</label>
          <input type="password" id="password" {...register("password")} className="w-full px-3 py-2 border border-[var(--color-border)] rounded-lg text-sm bg-[var(--color-bg)]" />
          {errors.password && <p className="text-xs mt-1" style={{ color: "#dc2626" }}>{errors.password.message}</p>}
        </div>
        <button type="submit" disabled={isPending} className="px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-sm font-medium cursor-pointer border-none disabled:opacity-60">
          {isPending ? "Creating account..." : "Register"}
        </button>
      </form>
    </>
  );
}
