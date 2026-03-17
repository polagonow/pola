import JSI from "@gojsx/jsi";

export default async function ProjectsPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await JSI.triggerError(searchParams.error || undefined);
  const projects = await JSI.getProjects();

  return (
    <div>
      <div className="section-badge">⬡ Projects</div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: "1.25rem",
        }}
      >
        <h1 style={{ fontSize: "1.5rem", fontWeight: 700 }}>Open source</h1>
        <span style={{ fontSize: ".85rem", color: "var(--muted)" }}>
          {projects.length} projects
        </span>
      </div>

      <div className="grid-2">
        {projects.map((p) => (
          <a
            key={p.id}
            href={`/projects/${p.id}`}
            style={{ textDecoration: "none", color: "inherit" }}
          >
            <div className="card">
              <div
                style={{
                  display: "flex",
                  alignItems: "flex-start",
                  justifyContent: "space-between",
                  gap: ".5rem",
                  marginBottom: ".25rem",
                }}
              >
                <h2>{p.title}</h2>
                <span className={`status status-${p.status}`}>{p.status}</span>
              </div>
              <p>{p.description}</p>
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
                  marginTop: ".75rem",
                }}
              >
                <div className="tech-list" style={{ margin: 0 }}>
                  {p.tech.map((t) => (
                    <span key={t} className="tech">
                      {t}
                    </span>
                  ))}
                </div>
                <span style={{ fontSize: ".8rem", color: "var(--muted)" }}>
                  ★ {p.stars}
                </span>
              </div>
            </div>
          </a>
        ))}
      </div>
    </div>
  );
}
