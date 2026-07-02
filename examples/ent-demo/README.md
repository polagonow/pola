# ent-demo

API-only Pola app verifying the **Ent** repository path end-to-end: an ent
schema (`db/models/schema/task.go`) with its generated client
(`db/client/ent`), a per-entity generated repository (ent stays generated —
its typed codegen client has no generic surface), a service, and `/tasks`
routes. The generated interface still embeds the framework's
`repository.Repository[Task, uint]` contract, which the ent implementation
satisfies.

Generated with:

```
pola new ent-demo --api-only
pola generate model Task title:string done:bool   # ent schema + codegen
pola generate repository Task title:string done:bool
pola generate service Task
pola generate route Task GET,POST,PUT,DELETE --service=Task
```

Verify:

```
go test ./...   # includes real CRUD against an in-memory ent client
pola serve      # then: curl localhost:3000/tasks
```
