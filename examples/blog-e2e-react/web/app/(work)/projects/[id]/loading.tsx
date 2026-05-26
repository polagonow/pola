export default function Loading() {
  return (
    <div className="opacity-50">
      <div
        className="bg-[var(--color-border)] rounded-sm mb-3"
        style={{
          height: "1.8rem",
          width: "40%",
        }}
      />
      <div
        className="bg-[var(--color-border)] rounded-sm mb-6"
        style={{
          height: ".8rem",
          width: "70%",
        }}
      />
      <div
        className="bg-[var(--color-border)] rounded-sm"
        style={{
          height: ".75rem",
          width: "30%",
        }}
      />
    </div>
  );
}
