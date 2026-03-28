"use client";
import { useState } from "react";

interface Item {
  title: string;
  [key: string]: unknown;
}

interface Props {
  items: Item[];
  placeholder?: string;
  renderItem: (item: Item) => React.ReactNode;
}

export default function SearchFilter({
  items,
  placeholder = "Search…",
  renderItem,
}: Props) {
  const [query, setQuery] = useState("");
  const filtered = query
    ? items.filter((i) => i.title.toLowerCase().includes(query.toLowerCase()))
    : items;

  return (
    <div>
      <input
        className="w-full py-2 px-3 mb-4 border border-[var(--color-border)] rounded-lg text-sm bg-[var(--color-bg)] text-[var(--color-fg)]"
        type="search"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={placeholder}
      />
      {filtered.length === 0 ? (
        <p className="text-[var(--color-muted)] text-sm">
          No results for "{query}"
        </p>
      ) : (
        filtered.map((item, i) => <div key={i}>{renderItem(item)}</div>)
      )}
    </div>
  );
}
