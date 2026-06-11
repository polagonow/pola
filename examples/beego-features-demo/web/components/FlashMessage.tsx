import { Auth } from "@pola/actions";

export default async function FlashMessage() {
  const flashes = await Auth.getFlash();
  if (!flashes || Object.keys(flashes).length === 0) return null;

  return (
    <div className="mb-4">
      {Object.entries(flashes).map(([category, message]) => (
        <div
          key={category}
          className="px-4 py-3 rounded-lg text-sm mb-2"
          style={{
            background: category === "error" ? "#fef2f2" : "#f0fdf4",
            border: `1px solid ${category === "error" ? "#fecaca" : "#bbf7d0"}`,
            color: category === "error" ? "#dc2626" : "#16a34a",
          }}
        >
          {message}
        </div>
      ))}
    </div>
  );
}
