import { Blog } from "@pola/di";

export default async function PostsPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await Blog.triggerError(searchParams.error || undefined);
  const posts = await Blog.getPosts();
  const tag = searchParams?.tag;
  const filtered = tag ? posts.filter((p) => p.tags.includes(tag)) : posts;

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-2xl font-bold">
          {tag ? `Posts tagged "${tag}"` : "All posts"}
        </h1>
        <span className="text-sm text-[var(--color-muted)]">
          {filtered.length} posts
        </span>
      </div>

      <div className="grid gap-4">
        {filtered.map((post) => (
          <a
            key={post.slug}
            href={`/posts/${post.slug}`}
            className="no-underline text-inherit"
          >
            <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm">
              <div className="text-sm text-[var(--color-muted)] flex flex-wrap gap-3 mb-2">
                <span>{post.date}</span>
                <span>{post.readTime} min read</span>
                <span>{post.author}</span>
              </div>
              <h2>{post.title}</h2>
              <p>{post.excerpt}</p>
              <div className="flex flex-wrap gap-1.5 mt-2">
                {post.tags.map((t) => (
                  <span key={t} className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-full px-2.5 py-0.5 text-xs text-[var(--color-muted)]">
                    {t}
                  </span>
                ))}
              </div>
            </div>
          </a>
        ))}
      </div>

      {filtered.length === 0 && (
        <p className="text-[var(--color-muted)]">No posts found for tag "{tag}".</p>
      )}
    </div>
  );
}
