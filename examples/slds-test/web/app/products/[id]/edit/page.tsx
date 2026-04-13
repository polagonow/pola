import { ProductAction } from "@pola/actions";
import EditProductForm from "@/components/products/edit-form";

export default async function EditProductPage({
  params,
}: {
  params: { id: string };
}) {
  const id = parseInt(params.id, 10);
  const item = await ProductAction.get(id);

  return (
    <article className="slds-card">
      <div className="slds-card__header slds-grid">
        <header className="slds-media slds-media_center slds-has-flexi-truncate">
          <div className="slds-media__body">
            <h2 className="slds-card__header-title slds-text-heading_small">{`Edit Product #${params.id}`}</h2>
          </div>
        </header>
      </div>
      <div className="slds-card__body slds-card__body_inner">
        <EditProductForm id={id} initialData={item} />
      </div>
    </article>
  );
}
