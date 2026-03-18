import JSI from "@pola/jsi";

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
    await JSI.triggerError(searchParams.error || undefined);
  const slug = params?.slug;

  // /docs — index
  if (!slug || slug.length === 0) {
    return (
      <div>
        <div className="detail-header">
          <h1 style={{ fontSize: "1.8rem", fontWeight: 800 }}>Documentation</h1>
          <p style={{ color: "var(--muted)", marginTop: ".4rem" }}>
            Learn how to build with GoJSX — Go-powered React Server Components.
          </p>
        </div>
        <div className="grid-2">
          {Object.entries(SECTION_LABELS).map(([key, label]) => (
            <a
              key={key}
              href={`/docs/${key}`}
              style={{ textDecoration: "none", color: "inherit" }}
            >
              <div className="card">
                <h3 style={{ marginBottom: ".35rem" }}>{label}</h3>
                <p style={{ fontSize: ".9rem", color: "var(--muted)" }}>
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
      <div className="card rsc-err">
        <strong>Section not found</strong>
        <p style={{ fontSize: ".9rem", margin: ".5rem 0" }}>
          No section named <code>{section}</code>.
        </p>
        <a href="/docs" className="btn btn-outline">
          Back to docs
        </a>
      </div>
    );
  }

  // /docs/[section] — section index
  if (!page) {
    return (
      <div>
        <div className="detail-header">
          <a href="/docs" className="back-link">
            ← Docs
          </a>
          <h1
            style={{ fontSize: "1.6rem", fontWeight: 800, marginTop: ".5rem" }}
          >
            {SECTION_LABELS[section]}
          </h1>
          <p style={{ color: "var(--muted)", marginTop: ".4rem" }}>
            {sectionDocs.index}
          </p>
        </div>
        <div
          style={{ display: "flex", flexDirection: "column", gap: ".75rem" }}
        >
          {Object.entries(sectionDocs)
            .filter(([k]) => k !== "index")
            .map(([key, summary]) => (
              <a
                key={key}
                href={`/docs/${section}/${key}`}
                style={{ textDecoration: "none", color: "inherit" }}
              >
                <div
                  className="card"
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center",
                  }}
                >
                  <div>
                    <div
                      style={{
                        fontWeight: 600,
                        marginBottom: ".2rem",
                        textTransform: "capitalize",
                      }}
                    >
                      {key.replace(/-/g, " ")}
                    </div>
                    <div style={{ fontSize: ".9rem", color: "var(--muted)" }}>
                      {summary}
                    </div>
                  </div>
                  <span style={{ color: "var(--muted)" }}>→</span>
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
      <div className="card rsc-err">
        <strong>Page not found</strong>
        <p style={{ fontSize: ".9rem", margin: ".5rem 0" }}>
          No page named <code>{page}</code> in <code>{section}</code>.
        </p>
        <a href={`/docs/${section}`} className="btn btn-outline">
          Back to {SECTION_LABELS[section]}
        </a>
      </div>
    );
  }

  return (
    <div>
      <div className="detail-header">
        <div className="meta" style={{ marginBottom: ".4rem" }}>
          <a
            href="/docs"
            style={{ color: "var(--muted)", textDecoration: "none" }}
          >
            Docs
          </a>
          <span>/</span>
          <a
            href={`/docs/${section}`}
            style={{ color: "var(--muted)", textDecoration: "none" }}
          >
            {SECTION_LABELS[section]}
          </a>
        </div>
        <h1
          style={{
            fontSize: "1.6rem",
            fontWeight: 800,
            textTransform: "capitalize",
          }}
        >
          {page.replace(/-/g, " ")}
        </h1>
      </div>
      <div className="card">
        <p>{content}</p>
      </div>
      <div
        style={{
          marginTop: "1.5rem",
          fontSize: ".85rem",
          color: "var(--muted)",
        }}
      >
        Path segments received: <code>{slug.join(" / ")}</code> (via{" "}
        <code>params.slug: string[]</code>)
      </div>
    </div>
  );
}
