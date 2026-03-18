import JSI from "@pola/di";

function Tag({ label }: { label: string }) {
  return <span className="tag">{label}</span>;
}

async function FeaturedPosts() {
  const posts = await JSI.getPosts();
  return (
    <div className="card-grid">
      {posts.slice(0, 2).map((post) => (
        <a
          key={post.slug}
          href={`/posts/${post.slug}`}
          style={{ textDecoration: "none", color: "inherit" }}
        >
          <div className="card">
            <div className="meta">
              <span>{post.date}</span>
              <span>{post.readTime} min read</span>
            </div>
            <h3>{post.title}</h3>
            <p>{post.excerpt}</p>
            <div className="tags">
              {post.tags.map((t) => (
                <Tag key={t} label={t} />
              ))}
            </div>
          </div>
        </a>
      ))}
    </div>
  );
}

async function FeaturedProjects() {
  const projects = await JSI.getProjects();
  return (
    <div className="grid-2">
      {projects.slice(0, 2).map((p) => (
        <a
          key={p.id}
          href={`/projects/${p.id}`}
          style={{ textDecoration: "none", color: "inherit" }}
        >
          <div className="card">
            <div
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                marginBottom: ".25rem",
              }}
            >
              <h3>{p.title}</h3>
              <span className={`status status-${p.status}`}>{p.status}</span>
            </div>
            <p>{p.description}</p>
            <div className="tech-list">
              {p.tech.map((t) => (
                <span key={t} className="tech">
                  {t}
                </span>
              ))}
            </div>
          </div>
        </a>
      ))}
    </div>
  );
}

export default async function HomePage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await JSI.triggerError(searchParams.error || undefined);
  return (
    <div>
      <div className="hero">
        <h1>
          Dev<span style={{ color: "var(--accent)" }}>Blog</span>
        </h1>
        <p>
          Writing about Go, React, and building dev tools. Server Components,
          esbuild, and the Goja VM.
        </p>
        <div className="hero-actions">
          <a href="/posts" className="btn btn-primary">
            Read posts
          </a>
          <a href="/projects" className="btn btn-outline">
            View projects
          </a>
        </div>
      </div>

      <div style={{ marginBottom: "2.5rem" }}>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: "1rem",
          }}
        >
          <h2 className="section-title" style={{ margin: 0 }}>
            Recent posts
          </h2>
          <a
            href="/posts"
            style={{ fontSize: ".85rem", color: "var(--muted)" }}
          >
            View all →
          </a>
        </div>
        <FeaturedPosts />
      </div>

      <div>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: "1rem",
          }}
        >
          <h2 className="section-title" style={{ margin: 0 }}>
            Projects
          </h2>
          <a
            href="/projects"
            style={{ fontSize: ".85rem", color: "var(--muted)" }}
          >
            View all →
          </a>
        </div>
        <FeaturedProjects />
      </div>
    </div>
  );
}
