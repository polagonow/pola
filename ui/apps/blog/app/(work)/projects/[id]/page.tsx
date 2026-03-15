import jsi from "../jsi";

export default async function ProjectPage({ params, searchParams }: { params: { id: string }, searchParams?: Record<string, string> }) {
  if (searchParams?.error !== undefined) await jsi.triggerError(searchParams.error || undefined);
  const p = await jsi.getProject(params.id);

  return (
    <div>
      <div className="detail-header">
        <div style={{ display: "flex", alignItems: "center", gap: ".75rem", marginBottom: ".75rem" }}>
          <h1 style={{ fontSize: "1.8rem", fontWeight: 800 }}>{p.title}</h1>
          <span className={`status status-${p.status}`}>{p.status}</span>
        </div>
        <p style={{ color: "var(--muted)", fontSize: "1rem" }}>{p.description}</p>
      </div>

      <div className="card" style={{ marginBottom: "1rem" }}>
        <dl style={{ display: "grid", gridTemplateColumns: "max-content 1fr", gap: ".5rem 1.5rem" }}>
          <dt style={{ fontWeight: 600, color: "var(--muted)", fontSize: ".85rem" }}>Stars</dt>
          <dd>★ {p.stars}</dd>
          <dt style={{ fontWeight: 600, color: "var(--muted)", fontSize: ".85rem" }}>Status</dt>
          <dd><span className={`status status-${p.status}`}>{p.status}</span></dd>
          <dt style={{ fontWeight: 600, color: "var(--muted)", fontSize: ".85rem" }}>Stack</dt>
          <dd>
            <div className="tech-list" style={{ margin: 0 }}>
              {p.tech.map(t => <span key={t} className="tech">{t}</span>)}
            </div>
          </dd>
        </dl>
      </div>
    </div>
  );
}
