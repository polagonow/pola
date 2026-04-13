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
      <div className="slds-grid slds-grid_align-spread slds-grid_vertical-align-center slds-m-bottom_medium">
        <h1 className="slds-text-heading_medium">Products</h1>
        <a href="/products/create" className="slds-button slds-button_brand">
          New Product
        </a>
      </div>

      <table className="slds-table slds-table_cell-buffer slds-table_bordered slds-table_striped">
        <thead>
          <tr className="slds-line-height_reset">
            <th scope="col"><div className="slds-truncate" title="ID">ID</div></th>
            <th scope="col"><div className="slds-truncate" title="Name">Name</div></th>
            <th scope="col"><div className="slds-truncate" title="Amount">Amount</div></th>
            <th scope="col" style={{ textAlign: "right" }}><div className="slds-truncate" title="Actions">Actions</div></th>
          </tr>
        </thead>
        <tbody>
          {result.items.map((item) => (
            <tr key={item.id}>
              <td><div className="slds-truncate">{item.id}</div></td>
              <td><div className="slds-truncate">{String(item.name ?? "")}</div></td>
              <td><div className="slds-truncate">{String(item.amount ?? "")}</div></td>
              <td style={{ textAlign: "right" }}>
                <div className="slds-button-group" role="group">
                  <a href={`/products/${item.id}`} className="slds-button slds-button_neutral">View</a>
                  <a href={`/products/${item.id}/edit`} className="slds-button slds-button_neutral">Edit</a>
                  <DeleteButton id={item.id} />
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {result.items.length === 0 && (
        <div className="slds-text-align_center slds-p-vertical_large slds-text-color_weak">
          No products found.
        </div>
      )}
    </div>
  );
}
