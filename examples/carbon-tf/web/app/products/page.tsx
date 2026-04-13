import { ProductAction } from "@pola/actions";
import DeleteButton from "@/components/products/delete-button";

export default async function ProductsPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  const page = parseInt(searchParams?.page || "1", 10);
  const perPage = parseInt(searchParams?.per_page || "25", 10);
  const result = await ProductAction.list(page, perPage);

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "1.5rem" }}>
        <h1 style={{ fontSize: "1.25rem", fontWeight: 600, margin: 0 }}>
          Products
        </h1>
        <a
          href="/products/create"
          className="cds--btn cds--btn--primary"
        >
          New Product
        </a>
      </div>

      <div className="cds--data-table-container">
        <table className="cds--data-table cds--data-table--compact">
          <thead>
            <tr>
              <th><span className="cds--table-header-label">ID</span></th>
              <th><span className="cds--table-header-label">Name</span></th>
              <th><span className="cds--table-header-label">Amount</span></th>
              <th style={{ textAlign: "right" }}><span className="cds--table-header-label">Actions</span></th>
            </tr>
          </thead>
          <tbody>
            {result.items.map((item) => (
              <tr key={item.id}>
                <td>{item.id}</td>
                <td>{String(item.name ?? "")}</td>
                <td>{String(item.amount ?? "")}</td>
                <td style={{ textAlign: "right" }}>
                  <div style={{ display: "inline-flex", gap: "0.5rem" }}>
                    <a href={`/products/${item.id}`} className="cds--btn cds--btn--ghost cds--btn--sm">View</a>
                    <a href={`/products/${item.id}/edit`} className="cds--btn cds--btn--ghost cds--btn--sm">Edit</a>
                    <DeleteButton id={item.id} />
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {result.items.length === 0 && (
        <p style={{ padding: "2rem 0", textAlign: "center", color: "#525252" }}>
          No products found.
        </p>
      )}
    </div>
  );
}
