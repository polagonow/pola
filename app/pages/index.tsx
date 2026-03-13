// app/pages/index.tsx
// This is a Server Component. It runs entirely in Go's Goja VM.
// No "use client" → never shipped to the browser.

// ---------------------------------------------------------------------------
// Sub-components (all server-side)
// ---------------------------------------------------------------------------

function ProductCard({ product }: { product: { id: number; name: string; price: number; stock: number } }) {
  return (
    <div className="product" key={String(product.id)}>
      <strong>{product.name}</strong>
      <span className="price">${product.price.toFixed(2)}</span>
      <small>({product.stock} in stock)</small>
    </div>
  );
}

function ProductList() {
  // ctx.getProducts() calls the Go function registered in main.go
  const products = ctx.getProducts();
  return (
    <section>
      <h2>Products <small style={{ fontSize: "0.75rem", color: "#888" }}>fetched via ctx.getProducts()</small></h2>
      <div className="product-list">
        {products.map((p) => (
          <ProductCard key={String(p.id)} product={p} />
        ))}
      </div>
    </section>
  );
}

function InfoBox() {
  return (
    <div style={{ background: "#e8f4fd", border: "1px solid #b3d9f7", borderRadius: "8px", padding: "1rem", marginBottom: "1.5rem" }}>
      <strong>⚡ Server-rendered with Go + Goja + RSC Flight Protocol</strong>
      <ul style={{ margin: "0.5rem 0 0", paddingLeft: "1.25rem", lineHeight: "1.8" }}>
        <li>This page rendered in a <strong>Goja JS VM</strong> inside Go</li>
        <li>Go functions are available as <code>ctx.fn()</code> and bare globals</li>
        <li>The result is streamed as RSC Flight chunks (<code>0:J{"{…}"}</code>)</li>
        <li>Client components (marked <code>use client</code>) are bundled by <strong>esbuild</strong></li>
        <li>Suspense boundaries stream their content as goroutines resolve</li>
      </ul>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page root – exported and registered as a route in main.go
// ---------------------------------------------------------------------------

export function IndexPage({ title }: { title?: string }) {
  return (
    <div className="page">
      <h1>{title ?? "GoJSX"}</h1>
      <InfoBox />
      <ProductList />
    </div>
  );
}
