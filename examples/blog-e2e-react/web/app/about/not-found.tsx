export default function AboutNotFound() {
  return (
    <div className="py-8">
      <h2>Not found</h2>
      <p className="text-[var(--color-muted)]">This section doesn't exist.</p>
      <a
        href="/"
        className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-transparent text-[var(--color-fg)] border-[var(--color-border)] hover:bg-[var(--color-surface)] hover:no-underline mt-4"
      >
        Go home
      </a>
    </div>
  );
}
