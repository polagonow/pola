import { Server } from "@pola/di";

// This is a React Server Component. It runs on the server inside Go's JS
// runtime. You can call Go functions directly using the @pola/di bridge.
//
// Actions are defined in the actions/ directory. Each Go struct becomes
// a named export: import { Server } from "@pola/di"

async function ServerInfo() {
  const info = await Server.getServerInfo();
  return (
    <div style={{
      background: "var(--color-surface)",
      border: "1px solid var(--color-border)",
      borderRadius: "var(--radius-lg)",
      padding: "1.25rem",
      fontSize: "0.875rem",
      fontFamily: "monospace",
    }}>
      <div style={{ fontWeight: 600, marginBottom: "0.5rem", fontFamily: "system-ui" }}>
        Server Info <span style={{ color: "var(--color-muted)", fontWeight: 400 }}>(from Go)</span>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "auto 1fr", gap: "0.25rem 1rem" }}>
        <span style={{ color: "var(--color-muted)" }}>Go</span>   <span>{info.goVersion}</span>
        <span style={{ color: "var(--color-muted)" }}>OS</span>   <span>{info.os}/{info.arch}</span>
        <span style={{ color: "var(--color-muted)" }}>Time</span> <span>{info.time}</span>
      </div>
    </div>
  );
}

async function Greeting() {
  const { message } = await Server.greet("test-app");
  return <p style={{ color: "var(--color-muted)", fontStyle: "italic" }}>{message}</p>;
}

export default async function HomePage() {
  return (
    <div>
      <div style={{ padding: "4rem 0 2rem" }}>
        <h1 style={{ fontSize: "2.5rem", fontWeight: 800, letterSpacing: "-0.02em" }}>
          Welcome to <span style={{ color: "var(--color-accent)" }}>test-app</span>
        </h1>
        <p style={{ color: "var(--color-muted)", fontSize: "1.125rem", marginTop: "0.75rem", maxWidth: 520 }}>
          Your Pola app is running. Edit <code>app/page.tsx</code> and <code>actions/server.go</code> to get started.
        </p>
        <Greeting />
      </div>

      <div style={{ marginBottom: "2rem" }}>
        <h2 style={{ fontSize: "1.25rem", fontWeight: 600, marginBottom: "0.75rem" }}>
          Go → React Bridge
        </h2>
        <p style={{ color: "var(--color-muted)", marginBottom: "1rem", fontSize: "0.9375rem" }}>
          The data below is fetched from Go functions defined in <code>actions/server.go</code>,
          called during server-side rendering via <code>{"await Server.getServerInfo()"}</code>.
        </p>
        <ServerInfo />
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: "1rem" }}>
        <Card
          title="Actions Bridge"
          description="Define Go structs in actions/ and call them from React: import { Server } from '@pola/di'."
        />
        <Card
          title="File-based Routing"
          description="Create app/about/page.tsx and it becomes /about. Supports layouts, loading states, and error boundaries."
        />
        <Card
          title="Single Binary"
          description="Run 'pola build' to compile your entire app — Go server, JS bundle, and static assets — into one binary."
        />
      </div>
    </div>
  );
}

function Card({ title, description }: { title: string; description: string }) {
  return (
    <div style={{
      border: "1px solid var(--color-border)",
      borderRadius: "var(--radius-lg)",
      padding: "1.5rem",
    }}>
      <h3 style={{ margin: 0, marginBottom: "0.5rem" }}>{title}</h3>
      <p style={{ margin: 0, color: "var(--color-muted)", fontSize: "0.875rem" }}>{description}</p>
    </div>
  );
}
