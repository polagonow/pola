import { Blog } from "@pola/actions";
import type { Metadata } from "@pola/core";

import LikeButton from "@/components/LikeButton";

export async function generateMetadata({
  params,
}: {
  params: { slug: string };
}): Promise<Metadata> {
  const post = await Blog.getPost(params.slug);

  return {
    title: post.title,
    description: post.excerpt,
    openGraph: {
      title: post.title,
      description: post.excerpt,
      type: "article",
    },
  };
}

export default async function PostPage({
  params,
  searchParams,
}: {
  params: { slug: string };
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await Blog.triggerError(searchParams.error);
  const post = await Blog.getPost(params.slug);

  return (
    <div>
      <div className="mb-6">
        <div className="text-sm text-[var(--color-muted)] flex flex-wrap gap-3 mb-3">
          <span>{post.date}</span>
          <span>{post.readTime} min read</span>
          <span>{post.author}</span>
        </div>
        <h1 className="text-3xl font-extrabold leading-tight">
          {post.title}
        </h1>
        <div className="flex flex-wrap gap-1.5 mt-3">
          {post.tags.map((t) => (
            <span key={t} className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-full px-2.5 py-0.5 text-xs text-[var(--color-muted)]">
              {t}
            </span>
          ))}
        </div>
      </div>

      <p className="text-[var(--color-muted)] text-[1.05rem] leading-relaxed mb-8">
        {post.excerpt}
      </p>

      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm mb-6">
        <p className="text-[var(--color-muted)] text-sm">
          <em>
            Full content would go here. This is a framework demo — the post body
            is not stored in the bridge data.
          </em>
        </p>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <LikeButton initialCount={42} />
          <span className="text-sm text-[var(--color-muted)]">
            Did you find this useful?
          </span>
        </div>
        <a
          href={`/posts/${params.slug}/revisions`}
          className="text-sm text-[var(--color-muted)]"
        >
          View revisions →
        </a>
      </div>
    </div>
  );
}
