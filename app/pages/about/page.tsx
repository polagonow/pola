// Pure synchronous server component — no async, no await, no data fetching.
// Used to verify that non-async RSC pages render correctly alongside
// async/Suspense-based pages.

export default function AboutPage({ version }: { version?: string }) {
  return (
    <div className="page">
      <h1>About</h1>
      <p>
        This is a <strong>synchronous</strong> Server Component — no{" "}
        <code>async</code>, no <code>await</code>, no data fetching.
      </p>
      <dl>
        <dt>Runtime</dt>
        <dd>Goja (Go JS VM)</dd>
        <dt>Renderer</dt>
        <dd>react-server-dom-webpack/server.browser</dd>
        <dt>Version</dt>
        <dd>{version ?? "dev"}</dd>
      </dl>
    </div>
  );
}
