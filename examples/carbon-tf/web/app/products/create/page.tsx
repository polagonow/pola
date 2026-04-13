import CreateProductForm from "@/components/products/create-form";

export default function CreateProductPage() {
  return (
    <div className="cds--tile" style={{ maxWidth: 640 }}>
      <h2 style={{ fontSize: "1.25rem", fontWeight: 600, marginTop: 0, marginBottom: "1.5rem" }}>
        Create Product
      </h2>
      <CreateProductForm />
    </div>
  );
}
