import { ProductAction } from "@pola/actions";
import DeleteButton from "@/components/products/delete-button";

export default async function ProductShowPage({
  params,
}: {
  params: { id: string };
}) {
  const item = await ProductAction.get(parseInt(params.id, 10));

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "1.5rem" }}>
        <h1 style={{ fontSize: "1.25rem", fontWeight: 600, margin: 0 }}>
          Product #{params.id}
        </h1>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <a
            href={`/products/${params.id}/edit`}
            className="cds--btn cds--btn--primary"
          >
            Edit
          </a>
          <DeleteButton id={item.id} />
        </div>
      </div>

      <div className="cds--structured-list">
        <div className="cds--structured-list-thead">
          <div className="cds--structured-list-row cds--structured-list-row--header-row">
            <div className="cds--structured-list-th" style={{ width: 192 }}>Field</div>
            <div className="cds--structured-list-th">Value</div>
          </div>
        </div>
        <div className="cds--structured-list-tbody">
          <div className="cds--structured-list-row">
            <div className="cds--structured-list-td" style={{ width: 192, fontWeight: 500, color: "#525252" }}>ID</div>
            <div className="cds--structured-list-td">{item.id}</div>
          </div>
          <div className="cds--structured-list-row">
            <div className="cds--structured-list-td" style={{ width: 192, fontWeight: 500, color: "#525252" }}>Name</div>
            <div className="cds--structured-list-td">{String(item.name ?? "")}</div>
          </div>
          <div className="cds--structured-list-row">
            <div className="cds--structured-list-td" style={{ width: 192, fontWeight: 500, color: "#525252" }}>Amount</div>
            <div className="cds--structured-list-td">{String(item.amount ?? "")}</div>
          </div>
        </div>
      </div>

      <div style={{ marginTop: "1.5rem" }}>
        <a href="/products" className="cds--btn cds--btn--ghost cds--btn--sm">
          ← Back to Products
        </a>
      </div>
    </div>
  );
}
