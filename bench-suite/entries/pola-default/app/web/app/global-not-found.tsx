export default function GlobalNotFound() {
  return (
    <div style={{ textAlign: "center", padding: "4rem 0" }}>
      <h1 style={{ fontSize: "3rem", fontWeight: 700 }}>404</h1>
      <p style={{ color: "var(--color-muted)" }}>This page does not exist</p>
      <a href="/" style={{ marginTop: "1rem", display: "inline-block" }}>
        Back to home
      </a>
    </div>
  );
}
