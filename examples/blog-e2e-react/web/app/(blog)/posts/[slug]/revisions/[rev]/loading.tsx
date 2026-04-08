export default function Loading() {
  return (
    <div className="animate-[pulse_1.4s_ease-in-out_infinite]">
      <div
        className="bg-[var(--color-border)] rounded-sm"
        style={{
          height: ".75rem",
          width: "30%",
          marginBottom: ".6rem",
        }}
      />
      <div
        className="bg-[var(--color-border)] rounded-sm mb-6"
        style={{
          height: "1.6rem",
          width: "70%",
        }}
      />
      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm flex flex-col gap-2">
        <div
          className="bg-[var(--color-border)] rounded-sm"
          style={{
            height: ".75rem",
            width: "25%",
          }}
        />
        <div
          className="bg-[var(--color-border)] rounded-sm"
          style={{
            height: "1rem",
            width: "80%",
          }}
        />
      </div>
    </div>
  );
}
