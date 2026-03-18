import JSI from "@pola/di";

export default async function RevisionPage({
  params,
  searchParams,
}: {
  params: { slug: string; rev: string };
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await JSI.triggerError(searchParams.error || undefined);
  const [post, revision] = await Promise.all([
    JSI.getPost(params.slug),
    JSI.getRevision(params.slug, params.rev),
  ]);

  return (
    <div>
      <div className="detail-header">
        <div className="meta" style={{ marginBottom: ".5rem" }}>
          <span>Revision {revision.rev}</span>
          <span>{revision.date}</span>
        </div>
        <h1 style={{ fontSize: "1.6rem", fontWeight: 800, lineHeight: 1.25 }}>
          {post.title}
        </h1>
      </div>

      <div className="card" style={{ marginBottom: "1.5rem" }}>
        <p
          style={{
            fontWeight: 600,
            marginBottom: ".35rem",
            fontSize: ".9rem",
            color: "var(--muted)",
          }}
        >
          Change summary
        </p>
        <p>{revision.summary}</p>
      </div>

      <a href={`/posts/${params.slug}`} className="back-link">
        ← Back to post
      </a>
    </div>
  );
}
