export default function Loading() {
  return (
    <div className="grid gap-4">
      {[1, 2, 3].map((i) => (
        <div key={i} className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm opacity-50">
          <div
            className="bg-[var(--color-border)] rounded-sm mb-3"
            style={{
              height: ".75rem",
              width: "40%",
            }}
          />
          <div
            className="bg-[var(--color-border)] rounded-sm mb-2"
            style={{
              height: "1rem",
              width: "80%",
            }}
          />
          <div
            className="bg-[var(--color-border)] rounded-sm"
            style={{
              height: ".8rem",
              width: "60%",
            }}
          />
        </div>
      ))}
    </div>
  );
}
