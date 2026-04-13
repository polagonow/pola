import { Server } from "@pola/actions";
import { Card, Row, Col } from "antd";

async function ServerInfo() {
  const info = await Server.getServerInfo();
  return (
    <Card size="small" style={{ fontFamily: "monospace", fontSize: "0.875rem" }}>
      <div style={{ marginBottom: 8, fontWeight: 600 }}>
        Server Info <span style={{ fontWeight: 400, color: "rgba(0, 0, 0, 0.45)" }}>(from Go)</span>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "auto 1fr", gap: "4px 16px" }}>
        <span style={{ color: "rgba(0, 0, 0, 0.45)" }}>Go</span>   <span>{info.goVersion}</span>
        <span style={{ color: "rgba(0, 0, 0, 0.45)" }}>OS</span>   <span>{info.os}/{info.arch}</span>
        <span style={{ color: "rgba(0, 0, 0, 0.45)" }}>Time</span> <span>{info.time}</span>
      </div>
    </Card>
  );
}

async function Greeting() {
  const { message } = await Server.greet("antd-test");
  return <p style={{ fontStyle: "italic", color: "rgba(0, 0, 0, 0.45)" }}>{message}</p>;
}

export default async function HomePage() {
  return (
    <div style={{ maxWidth: 1000, margin: "0 auto", padding: "0 24px" }}>
      <div style={{ paddingBottom: 32, paddingTop: 64 }}>
        <h1 style={{ fontSize: "2.5rem", fontWeight: 800, letterSpacing: "-0.02em", margin: 0 }}>
          Welcome to <span style={{ color: "#1677ff" }}>antd-test</span>
        </h1>
        <p style={{ marginTop: 12, maxWidth: 520, fontSize: "1.125rem", color: "rgba(0, 0, 0, 0.45)" }}>
          Your Pola app is running. Edit <code style={{ background: "#f5f5f5", border: "1px solid #d9d9d9", borderRadius: 4, padding: "2px 6px", fontSize: "0.875rem" }}>app/page.tsx</code> and <code style={{ background: "#f5f5f5", border: "1px solid #d9d9d9", borderRadius: 4, padding: "2px 6px", fontSize: "0.875rem" }}>actions/server.go</code> to get started.
        </p>
        <Greeting />
      </div>

      <div style={{ marginBottom: 32 }}>
        <h2 style={{ fontSize: "1.25rem", fontWeight: 600, marginBottom: 12 }}>
          Go → React Bridge
        </h2>
        <p style={{ marginBottom: 16, fontSize: "0.9375rem", color: "rgba(0, 0, 0, 0.45)" }}>
          The data below is fetched from Go functions defined in <code style={{ background: "#f5f5f5", border: "1px solid #d9d9d9", borderRadius: 4, padding: "2px 6px", fontSize: "0.875rem" }}>actions/server.go</code>,
          called during server-side rendering via <code style={{ background: "#f5f5f5", border: "1px solid #d9d9d9", borderRadius: 4, padding: "2px 6px", fontSize: "0.875rem" }}>{"await Server.getServerInfo()"}</code>.
        </p>
        <ServerInfo />
      </div>

      <Row gutter={16}>
        {[
          { title: "Actions Bridge", description: "Define Go structs in actions/ and call them from React: import { Server } from '@pola/actions'." },
          { title: "File-based Routing", description: "Create app/about/page.tsx and it becomes /about. Supports layouts, loading states, and error boundaries." },
          { title: "Single Binary", description: "Run 'pola build' to compile your entire app — Go server, JS bundle, and static assets — into one binary." },
        ].map((card) => (
          <Col key={card.title} xs={24} sm={8}>
            <Card style={{ height: "100%" }}>
              <h3 style={{ margin: "0 0 8px", fontWeight: 600 }}>{card.title}</h3>
              <p style={{ margin: 0, fontSize: "0.875rem", color: "rgba(0, 0, 0, 0.45)" }}>{card.description}</p>
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
}
