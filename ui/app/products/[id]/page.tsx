import jsi from "@/jsi";
import type { PageProps } from "@/types/page";

export default async function ProductDetailPage({ params }: PageProps) {
  const product = await jsi.getProduct(params.id);

  return (
    <div className="page">
      <a href="/products">← Back to Products</a>
      <h1>{product.name}</h1>
      <dl>
        <dt>ID</dt>       <dd>{product.id}</dd>
        <dt>Price</dt>    <dd>${(product.price as number).toFixed(2)}</dd>
        <dt>In stock</dt> <dd>{product.stock}</dd>
      </dl>
    </div>
  );
}
