import jsi from "@/jsi";

export default async function RevisionPage({
  params,
}: {
  params: { slug: string; rev: string };
}) {
  if (__request__.query.includes('error')) await jsi.triggerError(
    __request__.query.match(/error=([^&]*)/)?.[1] || 'Forced error'
  );
  const [post, revision] = await Promise.all([
    jsi.getPost(params.slug),
    jsi.getRevision(params.slug, params.rev),
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
        <p style={{ fontWeight: 600, marginBottom: ".35rem", fontSize: ".9rem", color: "var(--muted)" }}>
          Change summary
        </p>
        <p>{revision.summary}</p>
      </div>

      <a href={`/posts/${params.slug}`} className="back-link">← Back to post</a>
    </div>
  );
}
