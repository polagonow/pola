"use client"
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

export default function SearchFilter({ items, placeholder = "Search…", renderItem }: Props) {
  const [query, setQuery] = useState("");
  const filtered = query
    ? items.filter(i => i.title.toLowerCase().includes(query.toLowerCase()))
    : items;

  return (
    <div>
      <input
        className="search-input"
        type="search"
        value={query}
        onChange={e => setQuery(e.target.value)}
        placeholder={placeholder}
        style={{
          width: "100%", padding: ".5rem .75rem", marginBottom: "1rem",
          border: "1px solid var(--border)", borderRadius: "var(--radius)",
          fontSize: ".9rem", background: "var(--bg)", color: "var(--fg)",
        }}
      />
      {filtered.length === 0
        ? <p style={{ color: "var(--muted)", fontSize: ".9rem" }}>No results for "{query}"</p>
        : filtered.map((item, i) => <div key={i}>{renderItem(item)}</div>)
      }
    </div>
  );
}
