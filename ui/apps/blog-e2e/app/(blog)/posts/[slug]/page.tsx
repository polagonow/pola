import JSI from "@pola/jsi";

import LikeButton from "@/components/LikeButton";

export default async function PostPage({
  params,
  searchParams,
}: {
  params: { slug: string };
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await JSI.triggerError(searchParams.error || undefined);
  const post = await JSI.getPost(params.slug);

  return (
    <div>
      <div className="detail-header">
        <div className="meta" style={{ marginBottom: ".75rem" }}>
          <span>{post.date}</span>
          <span>{post.readTime} min read</span>
          <span>{post.author}</span>
        </div>
        <h1 style={{ fontSize: "2rem", fontWeight: 800, lineHeight: 1.2 }}>
          {post.title}
        </h1>
        <div className="tags" style={{ marginTop: ".75rem" }}>
          {post.tags.map((t) => (
            <span key={t} className="tag">
              {t}
            </span>
          ))}
        </div>
      </div>

      <p
        style={{
          color: "var(--muted)",
          fontSize: "1.05rem",
          lineHeight: 1.7,
          marginBottom: "2rem",
        }}
      >
        {post.excerpt}
      </p>

      <div className="card" style={{ marginBottom: "1.5rem" }}>
        <p style={{ color: "var(--muted)", fontSize: ".9rem" }}>
          <em>
            Full content would go here. This is a framework demo — the post body
            is not stored in the bridge data.
          </em>
        </p>
      </div>

      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
          <LikeButton initialCount={42} />
          <span style={{ fontSize: ".85rem", color: "var(--muted)" }}>
            Did you find this useful?
          </span>
        </div>
        <a
          href={`/posts/${params.slug}/revisions`}
          style={{ fontSize: ".85rem", color: "var(--muted)" }}
        >
          View revisions →
        </a>
      </div>
    </div>
  );
}
