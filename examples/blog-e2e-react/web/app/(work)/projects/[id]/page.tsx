import { Blog } from "@pola/actions";

const statusClasses: Record<string, string> = {
  active: "bg-green-100 text-green-800",
  stable: "bg-blue-100 text-blue-800",
  beta: "bg-yellow-100 text-yellow-800",
};

export default async function ProjectPage({
  params,
  searchParams,
}: {
  params: { id: string };
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await Blog.triggerError(searchParams.error);
  const p = await Blog.getProject(params.id);

  return (
    <div>
      <div className="mb-6">
        <div className="flex items-center gap-3 mb-3">
          <h1 className="text-3xl font-extrabold">{p.title}</h1>
          <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-semibold ${statusClasses[p.status] ?? ""}`}>{p.status}</span>
        </div>
        <p className="text-[var(--color-muted)] text-base">
          {p.description}
        </p>
      </div>

      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm mb-4">
        <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-2">
          <dt className="font-semibold text-[var(--color-muted)] text-sm">
            Stars
          </dt>
          <dd>★ {p.stars}</dd>
          <dt className="font-semibold text-[var(--color-muted)] text-sm">
            Status
          </dt>
          <dd>
            <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-semibold ${statusClasses[p.status] ?? ""}`}>{p.status}</span>
          </dd>
          <dt className="font-semibold text-[var(--color-muted)] text-sm">
            Stack
          </dt>
          <dd>
            <div className="flex flex-wrap gap-1.5 m-0">
              {p.tech.map((t) => (
                <span key={t} className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded px-2 py-0.5 text-xs font-mono">
                  {t}
                </span>
              ))}
            </div>
          </dd>
        </dl>
      </div>
    </div>
  );
}
