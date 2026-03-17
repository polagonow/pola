export default function Loading() {
  return (
    <div style={{ opacity: 0.5 }}>
      <div
        style={{
          height: "2rem",
          background: "var(--border)",
          borderRadius: 4,
          width: "70%",
          marginBottom: "1rem",
        }}
      />
      <div
        style={{
          height: ".8rem",
          background: "var(--border)",
          borderRadius: 4,
          width: "30%",
          marginBottom: "1.5rem",
        }}
      />
      {[100, 85, 92, 70].map((w, i) => (
        <div
          key={i}
          style={{
            height: ".75rem",
            background: "var(--border)",
            borderRadius: 4,
            width: `${w}%`,
            marginBottom: ".6rem",
          }}
        />
      ))}
    </div>
  );
}
