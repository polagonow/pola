export default function Loading() {
  return (
    <div style={{ animation: "pulse 1.4s ease-in-out infinite" }}>
      <div style={{ height: ".75rem", width: "30%", background: "var(--border)", borderRadius: 4, marginBottom: ".6rem" }} />
      <div style={{ height: "1.6rem", width: "70%", background: "var(--border)", borderRadius: 4, marginBottom: "1.5rem" }} />
      <div className="card" style={{ display: "flex", flexDirection: "column", gap: ".5rem" }}>
        <div style={{ height: ".75rem", width: "25%", background: "var(--border)", borderRadius: 4 }} />
        <div style={{ height: "1rem", width: "80%", background: "var(--border)", borderRadius: 4 }} />
      </div>
    </div>
  );
}
