# server-actions-demo

A minimal Pola app demonstrating **RSC server actions** (`'use server'`) with the
`nativersc` renderer.

## What it shows

- `web/app/actions/todos.ts` is a `'use server'` module. Every exported function
  becomes a **server action**.
- `web/app/page.tsx` is a **server component**. It calls `getTodos()` directly —
  the real code runs server-side in the Goja VM.
- `web/app/todo-list.tsx` is a `'use client'` component. It imports
  `addTodo` / `toggleTodo` / `deleteTodo` / `clearCompleted` and calls them like
  normal functions. At build time those imports are rewritten into client
  *server references* that `POST` to `/_pola/action`; no server code ships to the
  browser.

## How it works

```
build:  scan for 'use server'  ─► server bundle registers the functions in
                                   globalThis.__POLA_SERVER_ACTIONS__["id:export"]
                                ─► client bundle replaces the module with
                                   createServerReference(...) stubs

call:   client  ──POST /_pola/action {id, export_name, args}──►  Go handler
        (CSRF origin check + arg validation) ─► Goja VM runs the action
        ─► { success, result, error, redirect } JSON ─► client updates state
```

## Run

```sh
cd web && pnpm install && cd ..
go run .            # pola generates plugin wiring on first run
```

Then open the printed URL and add/toggle/delete todos — each interaction is a
server action round-trip.

## Try the endpoint directly

```sh
curl -s -X POST http://localhost:PORT/_pola/action \
  -H 'Content-Type: application/json' \
  -H 'Origin: http://localhost:PORT' \
  -d '{"id":"app/actions/todos","export_name":"addTodo","args":["buy milk"]}'
# => {"success":true,"result":[...],"redirect":null}
```

Note the action `id` is the module path under the app dir without extension
(`app/actions/todos`).
