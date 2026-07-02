# ent-demo

API-only Pola app verifying the **Ent** repository path end-to-end: an ent
schema (`db/models/schema/task.go`) with its generated client
(`db/client/ent`), a slim generated repository that embeds the framework's
generic `repository/ent` implementation (reflection-bound to the client's
`Task` sub-client, field writes via ent's runtime-validated `ent.Mutation`
API), a service, and `/tasks` routes.

Generated with:

```
pola new ent-demo --api-only
pola generate model Task title:string done:bool   # db/models/schema/task.go + ent codegen + migration
pola db migrate                                   # apply migrations (dev.db)
pola generate repository Task title:string done:bool
pola generate service Task
pola generate route Task GET,POST,PUT,DELETE --service=Task
```

Verify:

```
go test ./...   # includes real CRUD against an in-memory ent client
pola serve      # then: curl localhost:3000/tasks
```
