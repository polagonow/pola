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
      <div className="slds-grid slds-grid_align-spread slds-grid_vertical-align-center slds-m-bottom_medium">
        <h1 className="slds-text-heading_medium">Product #{params.id}</h1>
        <div className="slds-button-group" role="group">
          <a href={`/products/${params.id}/edit`} className="slds-button slds-button_brand">
            Edit
          </a>
          <DeleteButton id={item.id} />
        </div>
      </div>

      <article className="slds-card">
        <div className="slds-card__body slds-card__body_inner">
          <dl className="slds-dl_horizontal">
            <dt className="slds-dl_horizontal__label slds-text-color_weak">ID</dt>
            <dd className="slds-dl_horizontal__detail">{item.id}</dd>
            <dt className="slds-dl_horizontal__label slds-text-color_weak">Name</dt>
            <dd className="slds-dl_horizontal__detail">{String(item.name ?? "")}</dd>
            <dt className="slds-dl_horizontal__label slds-text-color_weak">Amount</dt>
            <dd className="slds-dl_horizontal__detail">{String(item.amount ?? "")}</dd>
          </dl>
        </div>
      </article>

      <div className="slds-m-top_medium">
        <a href="/products" className="slds-button slds-button_neutral">
          ← Back to Products
        </a>
      </div>
    </div>
  );
}
