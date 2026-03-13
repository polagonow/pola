// Server Component — runs in Goja VM on the server, never sent to browser.
// It imports Counter and ThemeToggle which are "use client" components.
// The RSC Flight encoder emits an I: module reference row for each one,
// and the browser loads those chunks via react-server-dom-esm/client.
//
import Counter from "../components/Counter";
import ThemeToggle from "../components/ThemeToggle";

function ProductCard({ product }: { product: { id: number; name: string; price: number; stock: number } }) {
  return (
    <div className="product">
      <strong>{product.name}</strong>
      <span className="price">${product.price.toFixed(2)}</span>
      <small>({product.stock} in stock)</small>
    </div>
  );
}

async function ProductList() {
  const products = await ctx.getProducts();

  return (
    <section>
      <h2>Products <small style={{ fontSize: "0.75rem", color: "#888" }}>from ctx.getProducts()</small></h2>
      <div className="product-list">
        {products.map((p) => (
          <ProductCard key={String(p.id)} product={p} />
        ))}
      </div>
    </section>
  );
}

export function IndexPage({ title }: { title?: string }) {
  return (
    <div className="page">
      <h1>{title ?? "GoJSX — Go + Goja + RSC"}</h1>

      {/*
        ThemeToggle is a "use client" component.
        The server emits a Flight reference row pointing to its chunk URL.
        The browser loads the chunk and renders it with useState/useEffect intact.
      */}
      <div style={{ display: "flex", alignItems: "center", gap: "1rem", marginBottom: "1.5rem" }}>
        <ThemeToggle />
        <span style={{ fontSize: "0.85rem", color: "#666" }}>
          ← client component with <code>useState</code> + <code>useEffect</code>
        </span>
      </div>

      <div style={{ background: "#e8f4fd", border: "1px solid #b3d9f7", borderRadius: "8px", padding: "1rem", marginBottom: "1.5rem" }}>
        <strong>⚡ Server + Client components mixed</strong>
        <ul style={{ margin: "0.5rem 0 0", paddingLeft: "1.25rem", lineHeight: "1.8" }}>
          <li>This page is a <strong>Server Component</strong> — rendered in Goja VM inside Go</li>
          <li><code>Counter</code> and <code>ThemeToggle</code> are <strong>Client Components</strong> (<code>"use client"</code>)</li>
          <li>The Flight stream emits <code>I:</code> module refs for client chunks</li>
          <li>The browser loads those chunks and hydrates them with React state</li>
        </ul>
      </div>

      <ProductList />

      {/*
        Counter is a "use client" component — interactive, lives in the browser.
        Server passes initialCount and label as serialised props in the Flight row.
      */}
      <section style={{ marginTop: "1.5rem" }}>
        <h2>Interactive counters <small style={{ fontSize: "0.75rem", color: "#888" }}>client component</small></h2>
        <div style={{ display: "flex", gap: "1rem", flexWrap: "wrap" }}>
          <Counter initialCount={0} label="Likes" />
          <Counter initialCount={100} label="Views" />
          <Counter initialCount={-5} label="Score" />
        </div>
      </section>
    </div>
  );
}

