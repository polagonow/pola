import { ProductAction } from "@pola/actions";
import ProductsListView from "@/components/products/list-view";

export default async function ProductsPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  const page = parseInt(searchParams?.page || "1", 10);
  const perPage = parseInt(searchParams?.per_page || "25", 10);
  const result = await ProductAction.list(page, perPage);

  return <ProductsListView items={result.items} />;
}
