import { ProductAction } from "@pola/actions";
import ProductShowView from "@/components/products/show-view";

export default async function ProductShowPage({
  params,
}: {
  params: { id: string };
}) {
  const item = await ProductAction.get(parseInt(params.id, 10));

  return <ProductShowView item={item} id={params.id} />;
}
