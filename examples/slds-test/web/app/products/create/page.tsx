import CreateProductForm from "@/components/products/create-form";

export default function CreateProductPage() {
  return (
    <article className="slds-card">
      <div className="slds-card__header slds-grid">
        <header className="slds-media slds-media_center slds-has-flexi-truncate">
          <div className="slds-media__body">
            <h2 className="slds-card__header-title slds-text-heading_small">Create Product</h2>
          </div>
        </header>
      </div>
      <div className="slds-card__body slds-card__body_inner">
        <CreateProductForm />
      </div>
    </article>
  );
}
