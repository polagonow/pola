import { getTodos } from "./actions/todos";
import { TodoList } from "./todo-list";

// Server component. It calls the 'use server' getTodos() directly (real code runs
// here in the VM) and hands the initial list to the interactive client island.
export default async function HomePage() {
  const todos = await getTodos();
  return (
    <main style={{ maxWidth: 560, margin: "0 auto", padding: "48px 20px" }}>
      <h1 style={{ marginBottom: 4 }}>Pola Server Actions</h1>
      <p style={{ color: "#666", marginTop: 0 }}>
        A <code>'use server'</code> module whose exports are called from a client
        component over HTTP (<code>POST /_pola/action</code>).
      </p>
      <TodoList initial={todos} />
    </main>
  );
}
