"use client";

import { useState } from "react";
// The functions are imported just like normal — esbuild rewrites this 'use server'
// module into client server-references that POST to /_pola/action. The `type`
// import is erased at build time, so no server code reaches the browser.
import { addTodo, toggleTodo, deleteTodo, clearCompleted } from "./actions/todos";
import type { Todo } from "./actions/todos";

export function TodoList({ initial }: { initial: Todo[] }) {
  const [todos, setTodos] = useState<Todo[]>(initial);
  const [text, setText] = useState("");
  const [pending, setPending] = useState(false);

  async function run(fn: () => Promise<Todo[]>) {
    setPending(true);
    try {
      setTodos(await fn());
    } finally {
      setPending(false);
    }
  }

  return (
    <section style={{ marginTop: 24 }}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const t = text.trim();
          if (!t) return;
          setText("");
          run(() => addTodo(t));
        }}
        style={{ display: "flex", gap: 8 }}
      >
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Add a todo…"
          style={{ flex: 1, padding: "8px 10px", border: "1px solid #ccc", borderRadius: 6 }}
        />
        <button type="submit" disabled={pending} style={btn}>
          Add
        </button>
      </form>

      <ul style={{ listStyle: "none", padding: 0, marginTop: 16 }}>
        {todos.map((t) => (
          <li
            key={t.id}
            style={{ display: "flex", alignItems: "center", gap: 10, padding: "8px 0", borderBottom: "1px solid #eee" }}
          >
            <input type="checkbox" checked={t.completed} onChange={() => run(() => toggleTodo(t.id))} />
            <span style={{ flex: 1, textDecoration: t.completed ? "line-through" : "none", color: t.completed ? "#999" : "inherit" }}>
              {t.text}
            </span>
            <button onClick={() => run(() => deleteTodo(t.id))} style={{ ...btn, background: "#fff", color: "#c00", borderColor: "#e3a3a3" }}>
              ✕
            </button>
          </li>
        ))}
      </ul>

      <button onClick={() => run(() => clearCompleted())} disabled={pending} style={{ ...btn, marginTop: 12 }}>
        Clear completed
      </button>
    </section>
  );
}

const btn: React.CSSProperties = {
  padding: "8px 14px",
  border: "1px solid #2563eb",
  background: "#2563eb",
  color: "#fff",
  borderRadius: 6,
  cursor: "pointer",
};
