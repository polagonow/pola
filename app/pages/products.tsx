function ProductCard({ product }: { product: { id: number; name: string; price: number; stock: number } }) {
  return (
    <div className="product">
      <strong>{product.name}</strong>
      <span className="price">${product.price.toFixed(2)}</span>
      <small>({product.stock} in stock)</small>
    </div>
  );
}

export function ProductsPage({ category }: { category?: string }) {
  const products = ctx.getProducts();
  return (
    <div className="page">
      <h1>Products</h1>
      {category && <p style={{ color: "#666" }}>Category: <strong>{category}</strong></p>}
      <div className="product-list">
        {products.map(p => <ProductCard key={String(p.id)} product={p} />)}
      </div>
    </div>
  );
}
