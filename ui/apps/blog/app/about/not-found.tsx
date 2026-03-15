export default function AboutNotFound() {
  return (
    <div style={{ padding: "2rem 0" }}>
      <h2>Not found</h2>
      <p style={{ color: "var(--muted)" }}>This section doesn't exist.</p>
      <a href="/" className="btn btn-outline" style={{ marginTop: "1rem", display: "inline-block" }}>Go home</a>
    </div>
  );
}
