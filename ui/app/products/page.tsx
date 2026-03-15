// import { Suspense } from "react";
import jsi from '@/jsi'

function ProductCard({ product }: { product: { id: number; name: string; price: number; stock: number } }) {
  return (
    <div className="product">
      <a href={`/products/${product.id}`}><strong>{product.name}</strong></a>
      <span className="price">${product.price.toFixed(2)}</span>
      <small>({product.stock} in stock)</small>
    </div>
  );
}

// Separate component that does the async work
export default async function ProductList({ category }: { category?: string }) {
  const products = await jsi.getProducts(); // suspends here

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

// // Parent wraps the async child in Suspense
// export default function ProductsPage({ searchParams }: { searchParams?: Record<string, string> }) {
//   return (
//     <Suspense fallback={<p>Loading Products...</p>}>
//       <ProductList category={searchParams?.category} /> {/* ← this can suspend */}
//     </Suspense>
//   );
// }
