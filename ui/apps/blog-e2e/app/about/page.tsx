import JSI from "@pola/jsi";

export default async function AboutPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await JSI.triggerError(searchParams.error || undefined);
  return (
    <div>
      <h1
        style={{ fontSize: "1.8rem", fontWeight: 800, marginBottom: ".75rem" }}
      >
        About this site
      </h1>

      <p
        style={{
          color: "var(--muted)",
          lineHeight: 1.7,
          marginBottom: "1.5rem",
          maxWidth: "560px",
        }}
      >
        DevBlog is a framework showcase built with <strong>GoJSX</strong> — a
        Go-powered React SSR framework that runs Server Components inside a Goja
        VM and streams them via the Flight wire protocol.
      </p>

      <div className="card" style={{ marginBottom: "1rem" }}>
        <h2
          style={{ fontSize: "1rem", fontWeight: 600, marginBottom: ".75rem" }}
        >
          Framework stack
        </h2>
        <dl
          style={{
            display: "grid",
            gridTemplateColumns: "max-content 1fr",
            gap: ".4rem 1.5rem",
            fontSize: ".9rem",
          }}
        >
          <dt style={{ color: "var(--muted)", fontWeight: 600 }}>Runtime</dt>
          <dd>Goja — Go JS VM</dd>
          <dt style={{ color: "var(--muted)", fontWeight: 600 }}>Renderer</dt>
          <dd>react-server-dom-esm (Flight protocol)</dd>
          <dt style={{ color: "var(--muted)", fontWeight: 600 }}>Bundler</dt>
          <dd>esbuild (two-pass: server CJS + client ESM)</dd>
          <dt style={{ color: "var(--muted)", fontWeight: 600 }}>Language</dt>
          <dd>Go + TypeScript + React</dd>
        </dl>
      </div>

      <div className="card">
        <h2
          style={{ fontSize: "1rem", fontWeight: 600, marginBottom: ".75rem" }}
        >
          Features demonstrated
        </h2>
        <ul
          style={{
            fontSize: ".9rem",
            lineHeight: 2,
            color: "var(--muted)",
            paddingLeft: "1.25rem",
          }}
        >
          <li>
            Async Server Components with <code>Suspense</code>
          </li>
          <li>Nested layouts at every route segment</li>
          <li>
            Route groups — <code>(blog)</code> and <code>(work)</code>
          </li>
          <li>Client components with React hooks</li>
          <li>Error boundaries at multiple levels</li>
          <li>Loading skeleton UIs</li>
          <li>Not-found pages per route</li>
          <li>
            Dynamic routes with <code>[slug]</code> and <code>[id]</code>
          </li>
          <li>
            Go ↔ JS bridge (<code>jsi</code>)
          </li>
          <li>searchParams in server components</li>
        </ul>
      </div>
    </div>
  );
}
