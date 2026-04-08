import { ProductAction } from "@pola/actions";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import EditProductForm from "@/components/products/edit-form";

export default async function EditProductPage({
  params,
}: {
  params: { id: string };
}) {
  const id = parseInt(params.id, 10);
  const item = await ProductAction.get(id);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Edit Product #{params.id}</CardTitle>
      </CardHeader>
      <CardContent>
        <EditProductForm id={id} initialData={item} />
      </CardContent>
    </Card>
  );
}
