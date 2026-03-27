"use client";

import { useState } from "react";

export default function ContactForm({
  submitAction,
}: {
  submitAction: (name: string, email: string, message: string) => Promise<any>;
}) {
  const [state, setState] = useState<{ success?: boolean; name?: string } | null>(null);
  const [pending, setPending] = useState(false);

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setPending(true);
    try {
      const fd = new FormData(e.currentTarget);
      const result = await submitAction(
        fd.get("name") as string,
        fd.get("email") as string,
        fd.get("message") as string,
      );
      setState(result);
    } catch (err) {
      console.error("Action failed:", err);
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <input
        name="name"
        placeholder="Name"
        required
        style={{
          padding: "8px 12px",
          border: "1px solid var(--border)",
          borderRadius: 6,
          background: "var(--surface)",
          color: "var(--fg)",
        }}
      />
      <input
        name="email"
        type="email"
        placeholder="Email"
        required
        style={{
          padding: "8px 12px",
          border: "1px solid var(--border)",
          borderRadius: 6,
          background: "var(--surface)",
          color: "var(--fg)",
        }}
      />
      <textarea
        name="message"
        placeholder="Message"
        required
        rows={4}
        style={{
          padding: "8px 12px",
          border: "1px solid var(--border)",
          borderRadius: 6,
          background: "var(--surface)",
          color: "var(--fg)",
          resize: "vertical",
        }}
      />
      <button
        type="submit"
        disabled={pending}
        style={{
          padding: "10px 20px",
          border: "none",
          borderRadius: 6,
          background: "var(--accent, #646cff)",
          color: "#fff",
          cursor: pending ? "not-allowed" : "pointer",
          opacity: pending ? 0.7 : 1,
          fontWeight: 600,
        }}
      >
        {pending ? "Sending..." : "Send Message"}
      </button>
      {state?.success && (
        <p style={{ color: "#4caf50", fontWeight: 500, margin: 0 }}>
          Thanks, {state.name}! Your message has been received.
        </p>
      )}
    </form>
  );
}
