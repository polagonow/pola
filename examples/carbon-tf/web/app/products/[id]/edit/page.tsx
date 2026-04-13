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
    <div className="cds--tile" style={{ maxWidth: 640 }}>
      <h2 style={{ fontSize: "1.25rem", fontWeight: 600, marginTop: 0, marginBottom: "1.5rem" }}>
        {`Edit Product #${params.id}`}
      </h2>
      <EditProductForm id={id} initialData={item} />
    </div>
  );
}
