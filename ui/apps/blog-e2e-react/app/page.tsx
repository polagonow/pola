import { Blog } from "@pola/di";

const statusClasses: Record<string, string> = {
  active: "bg-green-100 text-green-800",
  stable: "bg-blue-100 text-blue-800",
  beta: "bg-yellow-100 text-yellow-800",
};

function Tag({ label }: { label: string }) {
  return <span className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-full px-2.5 py-0.5 text-xs text-[var(--color-muted)]">{label}</span>;
}

async function FeaturedPosts() {
  const posts = await Blog.getPosts();
  return (
    <div className="grid gap-4">
      {posts.slice(0, 2).map((post) => (
        <a
          key={post.slug}
          href={`/posts/${post.slug}`}
          className="no-underline text-inherit"
        >
          <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm">
            <div className="text-sm text-[var(--color-muted)] flex flex-wrap gap-3 mb-2">
              <span>{post.date}</span>
              <span>{post.readTime} min read</span>
            </div>
            <h3>{post.title}</h3>
            <p>{post.excerpt}</p>
            <div className="flex flex-wrap gap-1.5 mt-2">
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
  const projects = await Blog.getProjects();
  return (
    <div className="grid grid-cols-2 gap-4 max-sm:grid-cols-1">
      {projects.slice(0, 2).map((p) => (
        <a
          key={p.id}
          href={`/projects/${p.id}`}
          className="no-underline text-inherit"
        >
          <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm">
            <div className="flex items-center justify-between mb-1">
              <h3>{p.title}</h3>
              <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-semibold ${statusClasses[p.status] ?? ""}`}>{p.status}</span>
            </div>
            <p>{p.description}</p>
            <div className="flex flex-wrap gap-1.5 mt-2">
              {p.tech.map((t) => (
                <span key={t} className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded px-2 py-0.5 text-xs font-mono">
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
    await Blog.triggerError(searchParams.error || undefined);
  return (
    <div>
      <div className="py-12 pb-8">
        <h1 className="text-[2.5rem] font-extrabold tracking-tight leading-tight">
          Dev<span className="text-[var(--color-accent)]">Blog</span>
        </h1>
        <p className="text-[var(--color-muted)] text-lg mt-3 max-w-[520px]">
          Writing about Go, React, and building dev tools. Server Components,
          esbuild, and the Goja VM.
        </p>
        <div className="flex gap-3 mt-6">
          <a href="/posts" className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-[var(--color-accent)] text-[var(--color-accent-fg)] border-[var(--color-accent)] hover:opacity-90 hover:no-underline">
            Read posts
          </a>
          <a href="/projects" className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-transparent text-[var(--color-fg)] border-[var(--color-border)] hover:bg-[var(--color-surface)] hover:no-underline">
            View projects
          </a>
        </div>
      </div>

      <div className="mb-10">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold m-0">
            Recent posts
          </h2>
          <a
            href="/posts"
            className="text-sm text-[var(--color-muted)]"
          >
            View all →
          </a>
        </div>
        <FeaturedPosts />
      </div>

      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold m-0">
            Projects
          </h2>
          <a
            href="/projects"
            className="text-sm text-[var(--color-muted)]"
          >
            View all →
          </a>
        </div>
        <FeaturedProjects />
      </div>
    </div>
  );
}
