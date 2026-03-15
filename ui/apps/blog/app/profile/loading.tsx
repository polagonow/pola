export default function Loading() {
  return (
    <div className="profile-card" style={{ opacity: .5 }}>
      <div className="avatar" style={{ background: "var(--border)" }} />
      <div>
        <div style={{ height: "1.2rem", background: "var(--border)", borderRadius: 4, width: 140, marginBottom: ".5rem" }} />
        <div style={{ height: ".8rem", background: "var(--border)", borderRadius: 4, width: 100 }} />
      </div>
    </div>
  );
}
