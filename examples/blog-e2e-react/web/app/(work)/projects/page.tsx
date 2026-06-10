import { Blog } from "@pola/actions";
import type { Metadata } from "@pola/core";

export const metadata: Metadata = {
  title: "Projects",
  description: "Open source projects and tools.",
};

const statusClasses: Record<string, string> = {
  active: "bg-green-100 text-green-800",
  stable: "bg-blue-100 text-blue-800",
  beta: "bg-yellow-100 text-yellow-800",
};

export default async function ProjectsPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await Blog.triggerError(searchParams.error);
  const projects = await Blog.getProjects();

  return (
    <div>
      <div className="inline-flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-[var(--color-muted)] mb-5">⬡ Projects</div>
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-2xl font-bold">Open source</h1>
        <span className="text-sm text-[var(--color-muted)]">
          {projects.length} projects
        </span>
      </div>

      <div className="grid grid-cols-2 gap-4 max-sm:grid-cols-1">
        {projects.map((p) => (
          <a
            key={p.id}
            href={`/projects/${p.id}`}
            className="no-underline text-inherit"
          >
            <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm">
              <div className="flex items-start justify-between gap-2 mb-1">
                <h2>{p.title}</h2>
                <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-semibold ${statusClasses[p.status] ?? ""}`}>{p.status}</span>
              </div>
              <p>{p.description}</p>
              <div className="flex items-center justify-between mt-3">
                <div className="flex flex-wrap gap-1.5 m-0">
                  {p.tech.map((t) => (
                    <span key={t} className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded px-2 py-0.5 text-xs font-mono">
                      {t}
                    </span>
                  ))}
                </div>
                <span className="text-xs text-[var(--color-muted)]">
                  ★ {p.stars}
                </span>
              </div>
            </div>
          </a>
        ))}
      </div>
    </div>
  );
}
