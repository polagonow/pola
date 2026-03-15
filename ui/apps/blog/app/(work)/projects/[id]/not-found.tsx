export default function ProjectNotFound() {
  return (
    <div style={{ padding: "2rem 0" }}>
      <div style={{ fontSize: "2.5rem", marginBottom: ".5rem" }}>404</div>
      <h2 style={{ marginBottom: ".5rem" }}>Project not found</h2>
      <p style={{ color: "var(--muted)", marginBottom: "1rem" }}>
        This project doesn't exist or has been removed.
      </p>
      <a href="/projects" className="btn btn-outline">Browse all projects</a>
    </div>
  );
}
