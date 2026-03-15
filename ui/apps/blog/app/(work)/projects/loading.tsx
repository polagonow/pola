export default function Loading() {
  return (
    <div className="grid-2">
      {[1, 2, 3].map(i => (
        <div key={i} className="card" style={{ opacity: .5 }}>
          <div style={{ height: "1rem", background: "var(--border)", borderRadius: 4, width: "50%", marginBottom: ".75rem" }} />
          <div style={{ height: ".75rem", background: "var(--border)", borderRadius: 4, width: "85%", marginBottom: ".5rem" }} />
          <div style={{ height: ".75rem", background: "var(--border)", borderRadius: 4, width: "40%" }} />
        </div>
      ))}
    </div>
  );
}
