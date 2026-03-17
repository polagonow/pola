export default function Loading() {
  return (
    <div className="card-grid">
      {[1, 2, 3].map((i) => (
        <div key={i} className="card" style={{ opacity: 0.5 }}>
          <div
            style={{
              height: ".75rem",
              background: "var(--border)",
              borderRadius: 4,
              width: "40%",
              marginBottom: ".75rem",
            }}
          />
          <div
            style={{
              height: "1rem",
              background: "var(--border)",
              borderRadius: 4,
              width: "80%",
              marginBottom: ".5rem",
            }}
          />
          <div
            style={{
              height: ".8rem",
              background: "var(--border)",
              borderRadius: 4,
              width: "60%",
            }}
          />
        </div>
      ))}
    </div>
  );
}
