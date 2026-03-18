import JSI from "@pola/di";

export default async function PostsPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await JSI.triggerError(searchParams.error || undefined);
  const posts = await JSI.getPosts();
  const tag = searchParams?.tag;
  const filtered = tag ? posts.filter((p) => p.tags.includes(tag)) : posts;

  return (
    <div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: "1.25rem",
        }}
      >
        <h1 style={{ fontSize: "1.5rem", fontWeight: 700 }}>
          {tag ? `Posts tagged "${tag}"` : "All posts"}
        </h1>
        <span style={{ fontSize: ".85rem", color: "var(--muted)" }}>
          {filtered.length} posts
        </span>
      </div>

      <div className="card-grid">
        {filtered.map((post) => (
          <a
            key={post.slug}
            href={`/posts/${post.slug}`}
            style={{ textDecoration: "none", color: "inherit" }}
          >
            <div className="card">
              <div className="meta">
                <span>{post.date}</span>
                <span>{post.readTime} min read</span>
                <span>{post.author}</span>
              </div>
              <h2>{post.title}</h2>
              <p>{post.excerpt}</p>
              <div className="tags">
                {post.tags.map((t) => (
                  <span key={t} className="tag">
                    {t}
                  </span>
                ))}
              </div>
            </div>
          </a>
        ))}
      </div>

      {filtered.length === 0 && (
        <p style={{ color: "var(--muted)" }}>No posts found for tag "{tag}".</p>
      )}
    </div>
  );
}
