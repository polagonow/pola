export default function Loading() {
  return (
    <div className="opacity-50">
      <div
        className="bg-[var(--color-border)] rounded-sm mb-4"
        style={{
          height: "2rem",
          width: "70%",
        }}
      />
      <div
        className="bg-[var(--color-border)] rounded-sm mb-6"
        style={{
          height: ".8rem",
          width: "30%",
        }}
      />
      {[100, 85, 92, 70].map((w, i) => (
        <div
          key={i}
          className="bg-[var(--color-border)] rounded-sm"
          style={{
            height: ".75rem",
            width: `${w}%`,
            marginBottom: ".6rem",
          }}
        />
      ))}
    </div>
  );
}
