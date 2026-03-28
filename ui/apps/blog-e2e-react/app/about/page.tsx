import { Blog } from "@pola/di";

export default async function AboutPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await Blog.triggerError(searchParams.error || undefined);
  return (
    <div>
      <h1 className="text-3xl font-extrabold mb-3">
        About this site
      </h1>

      <p className="text-[var(--color-muted)] leading-relaxed mb-6 max-w-[560px]">
        DevBlog is a framework showcase built with <strong>GoJSX</strong> — a
        Go-powered React SSR framework that runs Server Components inside a Goja
        VM and streams them via the Flight wire protocol.
      </p>

      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm mb-4">
        <h2 className="text-base font-semibold mb-3">
          Framework stack
        </h2>
        <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5 text-sm">
          <dt className="text-[var(--color-muted)] font-semibold">Runtime</dt>
          <dd>Goja — Go JS VM</dd>
          <dt className="text-[var(--color-muted)] font-semibold">Renderer</dt>
          <dd>react-server-dom-esm (Flight protocol)</dd>
          <dt className="text-[var(--color-muted)] font-semibold">Bundler</dt>
          <dd>esbuild (two-pass: server CJS + client ESM)</dd>
          <dt className="text-[var(--color-muted)] font-semibold">Language</dt>
          <dd>Go + TypeScript + React</dd>
        </dl>
      </div>

      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm">
        <h2 className="text-base font-semibold mb-3">
          Features demonstrated
        </h2>
        <ul className="text-sm leading-loose text-[var(--color-muted)] pl-5">
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
