import jsi from "@/jsi";

export default async function RevisionsPage({ params }: { params: { slug: string } }) {
  const [post, revisions] = await Promise.all([
    jsi.getPost(params.slug),
    jsi.getRevisions(params.slug),
  ]);

  return (
    <div>
      <div className="detail-header">
        <h1 style={{ fontSize: "1.4rem", fontWeight: 800 }}>Revision history</h1>
        <p style={{ color: "var(--muted)", fontSize: ".9rem", marginTop: ".25rem" }}>{post.title}</p>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: ".75rem" }}>
        {revisions.map(r => (
          <a
            key={r.rev}
            href={`/posts/${params.slug}/revisions/${r.rev}`}
            style={{ textDecoration: "none", color: "inherit" }}
          >
            <div className="card" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <div>
                <div style={{ fontWeight: 600, marginBottom: ".2rem" }}>{r.rev}</div>
                <div style={{ fontSize: ".9rem", color: "var(--muted)" }}>{r.summary}</div>
              </div>
              <span className="meta">{r.date}</span>
            </div>
          </a>
        ))}
      </div>

      <a href={`/posts/${params.slug}`} className="back-link" style={{ marginTop: "1.5rem", display: "inline-block" }}>
        ← Back to post
      </a>
    </div>
  );
}
