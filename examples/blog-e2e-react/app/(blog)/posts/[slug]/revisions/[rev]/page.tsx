import { Blog } from "@pola/actions";

export default async function RevisionPage({
  params,
  searchParams,
}: {
  params: { slug: string; rev: string };
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await Blog.triggerError(searchParams.error || undefined);
  const [post, revision] = await Promise.all([
    Blog.getPost(params.slug),
    Blog.getRevision(params.slug, params.rev),
  ]);

  return (
    <div>
      <div className="mb-6">
        <div className="text-sm text-[var(--color-muted)] flex flex-wrap gap-3 mb-2">
          <span>Revision {revision.rev}</span>
          <span>{revision.date}</span>
        </div>
        <h1 className="text-2xl font-extrabold leading-tight">
          {post.title}
        </h1>
      </div>

      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm mb-6">
        <p className="font-semibold mb-1 text-sm text-[var(--color-muted)]">
          Change summary
        </p>
        <p>{revision.summary}</p>
      </div>

      <a href={`/posts/${params.slug}`} className="text-sm text-[var(--color-muted)] mb-4 inline-block hover:text-[var(--color-fg)]">
        ← Back to post
      </a>
    </div>
  );
}
