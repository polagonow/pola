# Recipes: end-to-end walkthroughs

Three complete flows. Each is grounded in the framework's own example apps
(`examples/blog-e2e-react`, `examples/antd-test`, `examples/mcp-hello`).

---

## Recipe 1 — A blog CRUD with `scaffold`

Goal: a `Post` resource with list/show/create/edit pages and a working data layer.

```bash
pola new blog --css tailwind            # or: --ui shadcn (also pulls tailwind)
cd blog

pola generate scaffold Post \
  title:string:index \
  body:text \
  published:bool \
  author:references

pola db migrate                          # applies the auto-generated migration (sqlite)
pola dev                                 # http://localhost:3000/posts
```

What `scaffold Post …` writes:

- `db/models/gorm/post.go` — gorm model (`gorm.Model` + fields)
- `repositories/post_repository.go` + `repositories/gorm/post_repository.go` (+ `pagination.go`)
- `services/post_service.go`
- `actions/post_action.go` — the bridge: `PostAction` with `List`, `Get`, etc.
- `routes/posts/route.go` — JSON endpoint (optional alternative to the action)
- `web/schemas/post.ts` — Zod schema
- `web/app/posts/page.tsx`, `web/app/posts/[id]/page.tsx`,
  `web/app/posts/create/page.tsx`, `web/app/posts/[id]/edit/page.tsx`
  + `web/components/posts/{list-view,create-form,edit-form,delete-button}.tsx`
- a migration in `db/migrations/`

The generated list page calls the bridge from a Server Component:
```tsx
import { PostAction } from "@pola/actions";
import PostsListView from "@/components/posts/list-view";

export default async function PostsPage({ searchParams }: { searchParams?: Record<string,string> }) {
  const page = parseInt(searchParams?.page || "1", 10);
  const result = await PostAction.list(page, 25);   // camelCased, async
  return <PostsListView items={result.items} />;
}
```

To extend it, edit `services/post_service.go` (business logic) or add a method to
`actions/post_action.go` and run `pola generate` to refresh the TS types. Add fields later with a
fresh model + `pola generate migration AddSlugToPosts` + `pola db migrate`.

> Tip: `--skip-route`, `--skip-views`, etc. trim the scaffold (e.g. an API-only resource:
> `pola generate scaffold Post title:string --skip-views`).

---

## Recipe 2 — Expose a service to an LLM as an MCP tool

Goal: the *same* Go service that powers your React pages, also reachable by an MCP client
(Claude Desktop, MCP Inspector, curl). Mirrors `examples/mcp-hello`.

```bash
pola generate mcp init                         # adds the mcp { } block to Polafile.hcl
pola generate scaffold Greeting message:string # model+repo+service+action+...
pola generate mcp tool Greeting                # DI tool in mcp/tools/greeting_tool.go
pola db migrate
pola dev                                        # MCP endpoint mounts at /mcp
```

`pola generate mcp init` adds:
```hcl
mcp {
  enabled   = true
  transport = "http"      # http (streamable) | sse (legacy) | stdio (subprocess)
  mount     = "/mcp"
}
```

The DI tool resolves the scaffolded service straight from the framework container:
```go
package tools

import (
    "context"
    "encoding/json"
    "fmt"

    sdk "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/polagonow/pola/core"

    "myapp/repositories"
    "myapp/services"
)

type GreetingTool struct{ svc *services.GreetingService }

func NewGreetingTool(r *core.Registry) *GreetingTool {
    return &GreetingTool{svc: core.MustInvoke[*services.GreetingService](r)}
}

func (t *GreetingTool) Tool() *sdk.Tool {
    return &sdk.Tool{
        Name:        "greeting",
        Description: "List greetings or create a new one.",
        InputSchema: map[string]any{
            "type":       "object",
            "properties": map[string]any{"create": map[string]any{"type": "string"}},
        },
    }
}

func (t *GreetingTool) Handle(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
    var args struct{ Create string `json:"create"` }
    if raw := req.Params.Arguments; raw != nil { _ = json.Unmarshal(raw, &args) }
    if args.Create != "" {
        g := &repositories.Greeting{Message: args.Create}
        if err := t.svc.Create(ctx, g); err != nil { return nil, fmt.Errorf("create greeting: %w", err) }
        return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: fmt.Sprintf("created #%d", g.ID)}}}, nil
    }
    page, err := t.svc.List(ctx, repositories.ListParams{PerPage: 50})
    if err != nil { return nil, fmt.Errorf("list greetings: %w", err) }
    return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: fmt.Sprintf("%d greeting(s)", len(page.Items))}}}, nil
}
```

How it gets wired: an `mcp` autoload (priority 500) scans `mcp/{tools,resources,prompts}` for
`New<Name>{Tool,Resource,Prompt}` constructors and emits plugins that (1) provide the value into DI
and (2) on `Start`, resolve `*mcp.Server` and call `server.AddTool/AddResource/AddPrompt`. The
`mcp.Plugin()` is placed ahead of logging/recovery/csrf so MCP requests bypass that middleware.
No manual registration. Use `--no-di` for a simpler `init()`-registered typed tool (the SDK derives
the JSON schema from typed In/Out structs).

Talk to it (HTTP transport):
```bash
# 1) initialize → capture the session id from the Mcp-Session-Id response header
curl -s -i -X POST http://localhost:3000/mcp \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
        "protocolVersion":"2025-06-18","capabilities":{},
        "clientInfo":{"name":"curl","version":"1"}}}'

# 2) call the tool (reuse $SID from above)
curl -s -X POST http://localhost:3000/mcp \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -H "Mcp-Session-Id: $SID" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{
        "name":"greeting","arguments":{"create":"hi from MCP"}}}'
```

---

## Recipe 3 — File uploads with storage

Goal: a `Document` resource with an uploaded file, stored on the local filesystem.

```bash
pola generate storage --driver fs --root uploads      # StorageBlob + StorageAttachment + storage block

pola generate scaffold Document \
  title:string \
  file:references{StorageBlob}                          # the blob ref triggers file-upload handling

pola db migrate
pola dev
```

- `pola generate storage` creates `StorageBlob` (key, filename, content_type, byte_size, checksum)
  and `StorageAttachment` (polymorphic join linking any record to blobs), their repositories, and
  adds the `storage { driver = "fs", root = "uploads" }` block to `Polafile.hcl`.
- A `references{StorageBlob}` field marks the resource as file-bearing, so the generated route in
  `routes/documents/` parses multipart form data and persists the blob.
- For cloud storage, regenerate with `--driver rclone --root myremote:bucket/path
  --config-path /etc/rclone/rclone.conf` (or set the `storage` block's `env "production"` override).
- If you enable `image_processing`, serve transformed versions of image blobs via the `/_pola/image`
  endpoint (e.g. `"/_pola/image?url=…&width=128&height=128&fit=cover"`) or the `ImageProcessing.processURL` bridge binding inside a
  Server Component.

---

## Cross-cutting reminders

- After touching any `actions/` file, run **`pola generate`** to refresh `@pola/actions` types.
- Server Components fetch via the bridge and `await`; `"use client"` components take data as props.
- `pola db …` runs against **sqlite** locally even when production uses postgres.
- Ship with `pola build -o bin/app` (add `CGO_ENABLED=0` when on goja/sobek for a fully static binary).
