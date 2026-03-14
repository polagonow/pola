import { Suspense } from "react";

function ProductCard({ product }: { product: { id: number; name: string; price: number; stock: number } }) {
  return (
    <div className="product">
      <strong>{product.name}</strong>
      <span className="price">${product.price.toFixed(2)}</span>
      <small>({product.stock} in stock)</small>
    </div>
  );
}

// Separate component that does the async work
async function ProductList({ category }: { category?: string }) {
  const products = await ctx.getProducts(); // suspends here
  return (
    <div className="page">
      <h1>Products</h1>
      {category && <p>Category: <strong>{category}</strong></p>}
      <div className="product-list">
        {products.map(p => <ProductCard key={String(p.id)} product={p} />)}
      </div>
    </div>
  );
}

// Parent wraps the async child in Suspense
export function ProductsPage({ category }: { category?: string }) {
  return (
    <Suspense fallback={<p>Loading...</p>}>
      <ProductList category={category} /> {/* ← this can suspend */}
    </Suspense>
  );
}
