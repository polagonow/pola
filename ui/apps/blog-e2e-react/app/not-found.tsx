export default function NotFound() {
  return (
    <div className="py-12 text-center">
      <div className="text-5xl mb-2">404</div>
      <h2 className="mb-2">Page not found</h2>
      <p className="text-[var(--color-muted)] mb-6">
        The page you're looking for doesn't exist.
      </p>
      <a href="/" className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-[var(--color-accent)] text-[var(--color-accent-fg)] border-[var(--color-accent)] hover:opacity-90 hover:no-underline">
        Go home
      </a>
    </div>
  );
}
