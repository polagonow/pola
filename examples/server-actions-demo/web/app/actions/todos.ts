"use server";

// Every exported function in a 'use server' module becomes a server action:
//   - On the server it runs for real (in the Goja VM) and may touch a DB,
//     filesystem, secrets, etc.
//   - In a client component, importing one of these gives you an async stub
//     that POSTs to /_pola/action and returns the server result.
//
// State here is a module-level array to keep the demo dependency-free; in a real
// app you'd call into a database via the injected Go services.

export interface Todo {
  id: string;
  text: string;
  completed: boolean;
}

let todos: Todo[] = [
  { id: "1", text: "Learn React Server Components", completed: true },
  { id: "2", text: "Build server actions in Pola", completed: false },
];
let nextID = 3;

export async function getTodos(): Promise<Todo[]> {
  return todos;
}

export async function addTodo(text: string): Promise<Todo[]> {
  const trimmed = (text || "").trim();
  if (trimmed) {
    todos = [...todos, { id: String(nextID++), text: trimmed, completed: false }];
  }
  return todos;
}

export async function toggleTodo(id: string): Promise<Todo[]> {
  todos = todos.map((t) => (t.id === id ? { ...t, completed: !t.completed } : t));
  return todos;
}

export async function deleteTodo(id: string): Promise<Todo[]> {
  todos = todos.filter((t) => t.id !== id);
  return todos;
}

export async function clearCompleted(): Promise<Todo[]> {
  todos = todos.filter((t) => !t.completed);
  return todos;
}
