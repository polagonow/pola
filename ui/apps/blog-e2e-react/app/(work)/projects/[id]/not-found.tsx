export default function ProjectNotFound() {
  return (
    <div className="py-8">
      <div className="text-4xl mb-2">404</div>
      <h2 className="mb-2">Project not found</h2>
      <p className="text-[var(--color-muted)] mb-4">
        This project doesn't exist or has been removed.
      </p>
      <a href="/projects" className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border border-transparent transition-all bg-transparent text-[var(--color-fg)] border-[var(--color-border)] hover:bg-[var(--color-surface)] hover:no-underline">
        Browse all projects
      </a>
    </div>
  );
}
