import { ArticleAction } from "@pola/actions";

import DeleteButton from "@/components/articles/delete-button";

export default async function ArticlesPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  const page = parseInt(searchParams?.page || "1", 10);
  const perPage = parseInt(searchParams?.per_page || "25", 10);
  const result = await ArticleAction.list(page, perPage);

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "1.5rem" }}>
        <h1 style={{ fontSize: "1.5rem", fontWeight: 700 }}>Articles</h1>
        <a
          href="/articles/create"
          style={{
            display: "inline-block",
            padding: "0.5rem 1rem",
            background: "var(--color-accent, #3b82f6)",
            color: "#fff",
            borderRadius: "var(--radius-md, 6px)",
            textDecoration: "none",
            fontSize: "0.875rem",
            fontWeight: 500,
          }}
        >
          New Article
        </a>
      </div>

      <div style={{ overflowX: "auto" }}>
        <table style={{ width: "100%", borderCollapse: "collapse", fontSize: "0.875rem" }}>
          <thead>
            <tr style={{ borderBottom: "2px solid var(--color-border, #e5e7eb)" }}>
              <th style={{ textAlign: "left", padding: "0.75rem 0.5rem" }}>ID</th>
              <th style={{ textAlign: "left", padding: "0.75rem 0.5rem" }}>Title</th>
              <th style={{ textAlign: "left", padding: "0.75rem 0.5rem" }}>Body</th>
              <th style={{ textAlign: "right", padding: "0.75rem 0.5rem" }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {result.items.map((item) => (
              <tr key={item.id} style={{ borderBottom: "1px solid var(--color-border, #e5e7eb)" }}>
                <td style={{ padding: "0.75rem 0.5rem" }}>{item.id}</td>
                <td style={{ padding: "0.75rem 0.5rem" }}>{String(item.title ?? "")}</td>
                <td style={{ padding: "0.75rem 0.5rem" }}>{String(item.body ?? "")}</td>
                <td style={{ padding: "0.75rem 0.5rem", textAlign: "right" }}>
                  <span style={{ display: "inline-flex", gap: "0.5rem" }}>
                    <a href={`/articles/${item.id}`} style={{ color: "var(--color-accent, #3b82f6)", textDecoration: "none" }}>
                      View
                    </a>
                    <a href={`/articles/${item.id}/edit`} style={{ color: "var(--color-accent, #3b82f6)", textDecoration: "none" }}>
                      Edit
                    </a>
                    <DeleteButton id={item.id} />
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {result.items.length === 0 && (
        <p style={{ color: "var(--color-muted, #6b7280)", textAlign: "center", padding: "2rem 0" }}>
          No articles found.
        </p>
      )}
    </div>
  );
}
