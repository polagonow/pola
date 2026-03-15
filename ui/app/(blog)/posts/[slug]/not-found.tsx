export default function PostNotFound() {
  return (
    <div style={{ padding: "2rem 0" }}>
      <div style={{ fontSize: "2.5rem", marginBottom: ".5rem" }}>404</div>
      <h2 style={{ marginBottom: ".5rem" }}>Post not found</h2>
      <p style={{ color: "var(--muted)", marginBottom: "1rem" }}>
        This post may have been moved or deleted.
      </p>
      <a href="/posts" className="btn btn-outline">Browse all posts</a>
    </div>
  );
}
