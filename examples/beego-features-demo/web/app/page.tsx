import { Auth } from "@pola/actions";

export const metadata = {
  title: { absolute: "Beego Features Demo" },
};

export default async function HomePage() {
  const isAuth = await Auth.isAuthenticated();
  const greeting = await Auth.greet("Developer");

  return (
    <div>
      <div className="py-8">
        <h1 className="text-3xl font-extrabold tracking-tight">
          Pola <span className="text-[var(--color-accent)]">Features</span> Demo
        </h1>
        <p className="text-[var(--color-muted)] text-lg mt-2">{greeting}</p>
      </div>

      <div className="grid gap-4">
        <div className="border border-[var(--color-border)] rounded-lg p-5">
          <h2 className="text-lg font-bold mb-1">🔐 Authentication</h2>
          <p className="text-sm text-[var(--color-muted)] mb-3">
            Session-based auth with bcrypt password hashing and flash messages.
          </p>
          {isAuth ? (
            <a href="/profile" className="text-sm font-medium">View Profile →</a>
          ) : (
            <div className="flex gap-3">
              <a href="/login" className="text-sm font-medium">Login →</a>
              <a href="/register" className="text-sm font-medium">Register →</a>
            </div>
          )}
        </div>

        <div className="border border-[var(--color-border)] rounded-lg p-5">
          <h2 className="text-lg font-bold mb-1">🌍 Internationalization</h2>
          <p className="text-sm text-[var(--color-muted)] mb-3">
            JSON/YAML locale files with named placeholders and locale detection.
          </p>
          <div className="flex gap-3 text-sm">
            <a href="/?lang=en">English</a>
            <a href="/?lang=fr">Français</a>
            <a href="/?lang=es">Español</a>
          </div>
        </div>

        <div className="border border-[var(--color-border)] rounded-lg p-5">
          <h2 className="text-lg font-bold mb-1">⚡ Rate Limiting</h2>
          <p className="text-sm text-[var(--color-muted)]">
            Token bucket rate limiter at 10 req/s with burst of 20. Try hitting the API endpoints rapidly.
          </p>
        </div>

        <div className="border border-[var(--color-border)] rounded-lg p-5">
          <h2 className="text-lg font-bold mb-1">✅ Validation</h2>
          <p className="text-sm text-[var(--color-muted)] mb-3">
            Server-side struct validation with structured JSON error responses.
          </p>
          <code className="text-xs">POST /contacts</code>
        </div>

      </div>
    </div>
  );
}
