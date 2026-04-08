import { Server } from "@pola/actions";

async function ServerInfo() {
  const info = await Server.getServerInfo();
  return (
    <div className="rounded-lg border bg-muted/50 p-5 font-mono text-sm">
      <div className="mb-2 font-sans font-semibold">
        Server Info <span className="font-normal text-muted-foreground">(from Go)</span>
      </div>
      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
        <span className="text-muted-foreground">Go</span>   <span>{info.goVersion}</span>
        <span className="text-muted-foreground">OS</span>   <span>{info.os}/{info.arch}</span>
        <span className="text-muted-foreground">Time</span> <span>{info.time}</span>
      </div>
    </div>
  );
}

async function Greeting() {
  const { message } = await Server.greet("my-app");
  return <p className="italic text-muted-foreground">{message}</p>;
}

export default async function HomePage() {
  return (
    <div>
      <div className="pb-8 pt-16">
        <h1 className="text-4xl font-extrabold tracking-tight">
          Welcome to <span className="text-primary">my-app</span>
        </h1>
        <p className="mt-3 max-w-[520px] text-lg text-muted-foreground">
          Your Pola app is running. Edit <code className="rounded border bg-muted px-1.5 py-0.5 text-sm">app/page.tsx</code> and <code className="rounded border bg-muted px-1.5 py-0.5 text-sm">actions/server.go</code> to get started.
        </p>
        <Greeting />
      </div>

      <div className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          Go → React Bridge
        </h2>
        <p className="mb-4 text-[0.9375rem] text-muted-foreground">
          The data below is fetched from Go functions defined in <code className="rounded border bg-muted px-1.5 py-0.5 text-sm">actions/server.go</code>,
          called during server-side rendering via <code className="rounded border bg-muted px-1.5 py-0.5 text-sm">{"await Server.getServerInfo()"}</code>.
        </p>
        <ServerInfo />
      </div>

      <div className="grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-4">
        <Card
          title="Actions Bridge"
          description="Define Go structs in actions/ and call them from React: import { Server } from '@pola/actions'."
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
    <div className="rounded-lg border p-6">
      <h3 className="mb-2 font-semibold">{title}</h3>
      <p className="text-sm text-muted-foreground">{description}</p>
    </div>
  );
}
