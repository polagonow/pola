"use client";

import { useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { articleSchema, type Article } from "@/schemas/article";
import { csrfToken } from "@/utils/csrf";

export default function CreateArticleForm() {
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<Article>({
    resolver: zodResolver(articleSchema),
  });

  function onSubmit(data: Article) {
    setError(null);

    startTransition(async () => {
      try {
        const res = await fetch("/articles/", {
          method: "POST",
          headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken() },
          body: JSON.stringify(data),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || res.statusText);
        }
        window.location.href = "/articles";
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      }
    });
  }

  return (
    <>
      {error && (
        <div
          style={{
            padding: "0.75rem 1rem",
            marginBottom: "1rem",
            background: "#fef2f2",
            border: "1px solid #fecaca",
            borderRadius: "var(--radius-md, 6px)",
            color: "#dc2626",
            fontSize: "0.875rem",
          }}
        >
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)}>
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          <div>
            <label
              htmlFor="title"
              style={{
                display: "block",
                marginBottom: "0.25rem",
                fontSize: "0.875rem",
                fontWeight: 500,
              }}
            >
              Title
            </label>
            <input
              type="text"
              id="title"
              {...register("title")}
              style={{
                width: "100%",
                padding: "0.5rem",
                border: "1px solid var(--color-border, #e5e7eb)",
                borderRadius: "var(--radius-md, 6px)",
                fontSize: "0.875rem",
              }}
            />
            {errors.title?.message && (
              <p style={{ color: "#dc2626", fontSize: "0.75rem", marginTop: "0.25rem" }}>
                {errors.title.message}
              </p>
            )}
          </div>
          <div>
            <label
              htmlFor="body"
              style={{
                display: "block",
                marginBottom: "0.25rem",
                fontSize: "0.875rem",
                fontWeight: 500,
              }}
            >
              Body
            </label>
            <textarea
              id="body"
              {...register("body")}
              rows={4}
              style={{
                width: "100%",
                padding: "0.5rem",
                border: "1px solid var(--color-border, #e5e7eb)",
                borderRadius: "var(--radius-md, 6px)",
                fontSize: "0.875rem",
              }}
            />
            {errors.body?.message && (
              <p style={{ color: "#dc2626", fontSize: "0.75rem", marginTop: "0.25rem" }}>
                {errors.body.message}
              </p>
            )}
          </div>
        </div>

        <div style={{ display: "flex", gap: "0.75rem", marginTop: "1.5rem" }}>
          <button
            type="submit"
            disabled={isPending}
            style={{
              padding: "0.5rem 1.5rem",
              background: "var(--color-accent, #3b82f6)",
              color: "#fff",
              border: "none",
              borderRadius: "var(--radius-md, 6px)",
              fontSize: "0.875rem",
              fontWeight: 500,
              cursor: isPending ? "not-allowed" : "pointer",
              opacity: isPending ? 0.6 : 1,
            }}
          >
            {isPending ? "Creating..." : "Create Article"}
          </button>
          <a
            href="/articles"
            style={{
              padding: "0.5rem 1.5rem",
              border: "1px solid var(--color-border, #e5e7eb)",
              borderRadius: "var(--radius-md, 6px)",
              textDecoration: "none",
              color: "inherit",
              fontSize: "0.875rem",
            }}
          >
            Cancel
          </a>
        </div>
      </form>
    </>
  );
}
