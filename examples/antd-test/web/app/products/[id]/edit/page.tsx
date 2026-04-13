import { ProductAction } from "@pola/actions";
import { Card } from "antd";
import EditProductForm from "@/components/products/edit-form";

export default async function EditProductPage({
  params,
}: {
  params: { id: string };
}) {
  const id = parseInt(params.id, 10);
  const item = await ProductAction.get(id);

  return (
    <Card title={`Edit Product #${params.id}`}>
      <EditProductForm id={id} initialData={item} />
    </Card>
  );
}
