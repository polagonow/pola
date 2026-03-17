export default function Loading() {
  return (
    <div style={{ opacity: 0.5 }}>
      <div
        style={{
          height: "1.8rem",
          background: "var(--border)",
          borderRadius: 4,
          width: "40%",
          marginBottom: ".75rem",
        }}
      />
      <div
        style={{
          height: ".8rem",
          background: "var(--border)",
          borderRadius: 4,
          width: "70%",
          marginBottom: "1.5rem",
        }}
      />
      <div
        style={{
          height: ".75rem",
          background: "var(--border)",
          borderRadius: 4,
          width: "30%",
        }}
      />
    </div>
  );
}
