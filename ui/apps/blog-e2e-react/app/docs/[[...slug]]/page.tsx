import di from "@pola/di";

const DOCS: Record<string, Record<string, string>> = {
  "getting-started": {
    index: "Everything you need to get up and running with Pola.",
    installation:
      "Install Go 1.22+, clone the repo, and run `go run ./example`.",
    configuration: "Configure appDir, bridge functions, and the VM pool size.",
  },
  "core-concepts": {
    index: "Understand the building blocks: RSC, Flight, and the Goja VM.",
    "server-components":
      "Server Components run in Go via Goja and stream Flight data.",
    "client-components":
      "Mark a file with `'use client'` to opt into browser rendering.",
    routing:
      "File-system routing: page.tsx, layout.tsx, error.tsx, loading.tsx.",
  },
  "api-reference": {
    index: "Full API reference for the Go server package and JSI bridge.",
    "bridge-config":
      "BridgeConfig wires Go functions into the JS runtime context.",
    "route-patterns":
      "Supports :param, [...catch-all], and [[...optional]] segments.",
  },
};

const SECTION_LABELS: Record<string, string> = {
  "getting-started": "Getting Started",
  "core-concepts": "Core Concepts",
  "api-reference": "API Reference",
};

export default async function DocsPage({
  params,
  searchParams,
}: {
  params?: { slug?: string[] };
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await di.triggerError(searchParams.error || undefined);
  const slug = params?.slug;

  // /docs — index
  if (!slug || slug.length === 0) {
    return (
      <div>
        <div className="mb-6">
          <h1 className="text-3xl font-extrabold">Documentation</h1>
          <p className="text-[var(--color-muted)] mt-1.5">
            Learn how to build with GoJSX — Go-powered React Server Components.
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4 max-sm:grid-cols-1">
          {Object.entries(SECTION_LABELS).map(([key, label]) => (
            <a
              key={key}
              href={`/docs/${key}`}
              className="no-underline text-inherit"
            >
              <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm">
                <h3 className="mb-1">{label}</h3>
                <p className="text-sm text-[var(--color-muted)]">
                  {DOCS[key].index}
                </p>
              </div>
            </a>
          ))}
        </div>
      </div>
    );
  }

  const [section, page] = slug;
  const sectionDocs = DOCS[section];

  if (!sectionDocs) {
    return (
      <div className="text-red-600 bg-red-50 px-4 py-3 rounded-lg border border-red-200">
        <strong>Section not found</strong>
        <p className="text-sm my-2">
          No section named <code>{section}</code>.
        </p>
        <a href="/docs" className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-transparent text-[var(--color-fg)] border-[var(--color-border)] hover:bg-[var(--color-surface)] hover:no-underline">
          Back to docs
        </a>
      </div>
    );
  }

  // /docs/[section] — section index
  if (!page) {
    return (
      <div>
        <div className="mb-6">
          <a href="/docs" className="text-sm text-[var(--color-muted)] mb-4 inline-block hover:text-[var(--color-fg)]">
            ← Docs
          </a>
          <h1 className="text-2xl font-extrabold mt-2">
            {SECTION_LABELS[section]}
          </h1>
          <p className="text-[var(--color-muted)] mt-1.5">
            {sectionDocs.index}
          </p>
        </div>
        <div className="flex flex-col gap-3">
          {Object.entries(sectionDocs)
            .filter(([k]) => k !== "index")
            .map(([key, summary]) => (
              <a
                key={key}
                href={`/docs/${section}/${key}`}
                className="no-underline text-inherit"
              >
                <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm flex justify-between items-center">
                  <div>
                    <div className="font-semibold mb-0.5 capitalize">
                      {key.replace(/-/g, " ")}
                    </div>
                    <div className="text-sm text-[var(--color-muted)]">
                      {summary}
                    </div>
                  </div>
                  <span className="text-[var(--color-muted)]">→</span>
                </div>
              </a>
            ))}
        </div>
      </div>
    );
  }

  // /docs/[section]/[page]
  const content = sectionDocs[page];
  if (!content) {
    return (
      <div className="text-red-600 bg-red-50 px-4 py-3 rounded-lg border border-red-200">
        <strong>Page not found</strong>
        <p className="text-sm my-2">
          No page named <code>{page}</code> in <code>{section}</code>.
        </p>
        <a href={`/docs/${section}`} className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-transparent text-[var(--color-fg)] border-[var(--color-border)] hover:bg-[var(--color-surface)] hover:no-underline">
          Back to {SECTION_LABELS[section]}
        </a>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6">
        <div className="text-sm text-[var(--color-muted)] flex flex-wrap gap-3 mb-1.5">
          <a
            href="/docs"
            className="text-[var(--color-muted)] no-underline"
          >
            Docs
          </a>
          <span>/</span>
          <a
            href={`/docs/${section}`}
            className="text-[var(--color-muted)] no-underline"
          >
            {SECTION_LABELS[section]}
          </a>
        </div>
        <h1 className="text-2xl font-extrabold capitalize">
          {page.replace(/-/g, " ")}
        </h1>
      </div>
      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm">
        <p>{content}</p>
      </div>
      <div className="mt-6 text-sm text-[var(--color-muted)]">
        Path segments received: <code>{slug.join(" / ")}</code> (via{" "}
        <code>params.slug: string[]</code>)
      </div>
    </div>
  );
}
