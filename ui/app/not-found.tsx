export default function NotFound() {
  return (
    <div style={{ padding: "3rem 0", textAlign: "center" }}>
      <div style={{ fontSize: "3rem", marginBottom: ".5rem" }}>404</div>
      <h2 style={{ marginBottom: ".5rem" }}>Page not found</h2>
      <p style={{ color: "var(--muted)", marginBottom: "1.5rem" }}>
        The page you're looking for doesn't exist.
      </p>
      <a href="/" className="btn btn-primary">Go home</a>
    </div>
  );
}
