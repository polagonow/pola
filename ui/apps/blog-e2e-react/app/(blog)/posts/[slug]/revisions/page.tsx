import { Blog } from "@pola/di";

export default async function RevisionsPage({
  params,
  searchParams,
}: {
  params: { slug: string };
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await Blog.triggerError(searchParams.error || undefined);
  const [post, revisions] = await Promise.all([
    Blog.getPost(params.slug),
    Blog.getRevisions(params.slug),
  ]);

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-xl font-extrabold">
          Revision history
        </h1>
        <p className="text-[var(--color-muted)] text-sm mt-1">
          {post.title}
        </p>
      </div>

      <div className="flex flex-col gap-3">
        {revisions.map((r) => (
          <a
            key={r.rev}
            href={`/posts/${params.slug}/revisions/${r.rev}`}
            className="no-underline text-inherit"
          >
            <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm flex justify-between items-center">
              <div>
                <div className="font-semibold mb-0.5">
                  {r.rev}
                </div>
                <div className="text-sm text-[var(--color-muted)]">
                  {r.summary}
                </div>
              </div>
              <span className="text-sm text-[var(--color-muted)] flex flex-wrap gap-3">{r.date}</span>
            </div>
          </a>
        ))}
      </div>

      <a
        href={`/posts/${params.slug}`}
        className="text-sm text-[var(--color-muted)] mt-6 inline-block hover:text-[var(--color-fg)]"
      >
        ← Back to post
      </a>
    </div>
  );
}
