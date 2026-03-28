export default function Loading() {
  return (
    <div className="flex gap-6 items-start opacity-50">
      <div className="w-[72px] h-[72px] rounded-full bg-[var(--color-border)] shrink-0" />
      <div>
        <div
          className="rounded bg-[var(--color-border)] mb-2"
          style={{ height: "1.2rem", width: 140 }}
        />
        <div
          className="rounded bg-[var(--color-border)]"
          style={{ height: ".8rem", width: 100 }}
        />
      </div>
    </div>
  );
}
