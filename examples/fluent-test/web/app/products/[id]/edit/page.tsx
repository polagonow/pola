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
    <div>
      <h2 style={{ fontSize: "1.25rem", fontWeight: 600, marginBottom: 16 }}>{`Edit Product #${params.id}`}</h2>
      <EditProductForm id={id} initialData={item} />
    </div>
  );
}
