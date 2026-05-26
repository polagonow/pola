import { Server } from "@pola/actions";

async function ServerInfo() {
  const info = await Server.getServerInfo();
  return (
    <div style={{ border: "1px solid #d8dde6", borderRadius: 4, padding: 20, fontFamily: "monospace", fontSize: "0.875rem" }}>
      <div style={{ marginBottom: 8, fontFamily: "'Salesforce Sans', Arial, sans-serif", fontWeight: 600 }}>
        Server Info <span style={{ fontWeight: 400, color: "#706e6b" }}>(from Go)</span>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "auto 1fr", gap: "4px 16px" }}>
        <span style={{ color: "#706e6b" }}>Go</span>   <span>{info.goVersion}</span>
        <span style={{ color: "#706e6b" }}>OS</span>   <span>{info.os}/{info.arch}</span>
        <span style={{ color: "#706e6b" }}>Time</span> <span>{info.time}</span>
      </div>
    </div>
  );
}

async function Greeting() {
  const { message } = await Server.greet("slds-test");
  return <p style={{ fontStyle: "italic", color: "#706e6b" }}>{message}</p>;
}

export default async function HomePage() {
  return (
    <div style={{ maxWidth: 1000, margin: "0 auto", padding: "0 24px" }}>
      <div style={{ paddingBottom: 32, paddingTop: 64 }}>
        <h1 style={{ fontSize: "2.5rem", fontWeight: 800, letterSpacing: "-0.02em", margin: 0 }}>
          Welcome to <span style={{ color: "#0070d2" }}>slds-test</span>
        </h1>
        <p style={{ marginTop: 12, maxWidth: 520, fontSize: "1.125rem", color: "#706e6b" }}>
          Your Pola app is running. Edit <code style={{ background: "#f3f3f3", border: "1px solid #d8dde6", borderRadius: 4, padding: "2px 6px", fontSize: "0.875rem" }}>app/page.tsx</code> and <code style={{ background: "#f3f3f3", border: "1px solid #d8dde6", borderRadius: 4, padding: "2px 6px", fontSize: "0.875rem" }}>actions/server.go</code> to get started.
        </p>
        <Greeting />
      </div>

      <div style={{ marginBottom: 32 }}>
        <h2 style={{ fontSize: "1.25rem", fontWeight: 600, marginBottom: 12 }}>
          Go → React Bridge
        </h2>
        <p style={{ marginBottom: 16, fontSize: "0.9375rem", color: "#706e6b" }}>
          The data below is fetched from Go functions defined in <code style={{ background: "#f3f3f3", border: "1px solid #d8dde6", borderRadius: 4, padding: "2px 6px", fontSize: "0.875rem" }}>actions/server.go</code>,
          called during server-side rendering via <code style={{ background: "#f3f3f3", border: "1px solid #d8dde6", borderRadius: 4, padding: "2px 6px", fontSize: "0.875rem" }}>{"await Server.getServerInfo()"}</code>.
        </p>
        <ServerInfo />
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: 16 }}>
        {[
          { title: "Actions Bridge", description: "Define Go structs in actions/ and call them from React: import { Server } from '@pola/actions'." },
          { title: "File-based Routing", description: "Create app/about/page.tsx and it becomes /about. Supports layouts, loading states, and error boundaries." },
          { title: "Single Binary", description: "Run 'pola build' to compile your entire app — Go server, JS bundle, and static assets — into one binary." },
        ].map((card) => (
          <div key={card.title} style={{ border: "1px solid #d8dde6", borderRadius: 4, padding: 24 }}>
            <h3 style={{ margin: "0 0 8px", fontWeight: 600 }}>{card.title}</h3>
            <p style={{ margin: 0, fontSize: "0.875rem", color: "#706e6b" }}>{card.description}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
