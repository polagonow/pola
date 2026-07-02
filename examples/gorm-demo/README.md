# gorm-demo

API-only Pola app verifying the **GORM** repository path end-to-end: a `Task`
entity whose generated repository embeds the framework's generic
`repository.Repository[Task, uint]` (see `repository/gorm` in the framework),
wired through a service into `/tasks` routes.

Generated with:

```
pola new gorm-demo --api-only
pola generate model Task title:string done:bool        # db/models/gorm/task.go + migration
pola db migrate                                         # apply migrations (dev.db)
pola generate repository Task title:string done:bool
pola generate service Task
pola generate route Task GET,POST,PUT,DELETE --service=Task
```

Verify:

```
go test ./...   # includes real CRUD against in-memory SQLite
pola serve      # then: curl localhost:3000/tasks
```
