export default function RevisionNotFound() {
  return (
    <div className="text-red-600 bg-red-50 px-4 py-3 rounded-lg border border-red-200">
      <strong>Revision not found</strong>
      <p className="text-sm my-2">
        That revision does not exist for this post.
      </p>
      <a className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-transparent text-[var(--color-fg)] border-[var(--color-border)] hover:bg-[var(--color-surface)] hover:no-underline" href="/posts">
        Back to posts
      </a>
    </div>
  );
}
