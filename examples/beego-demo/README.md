# beego-demo

API-only Pola app verifying the **Beego ORM** repository path end-to-end: a
`Task` entity (with generated `orm:"column(...)"` tags and `TableName()`)
whose repository embeds the framework's generic
`repository.Repository[Task, uint]` (see `repository/beego` in the framework),
wired through a service into `/tasks` routes.

Generated with:

```
pola new beego-demo --api-only
pola generate repository Task title:string done:bool
pola generate service Task
pola generate route Task GET,POST,PUT,DELETE --service=Task
```

Verify:

```
go test ./...   # includes real CRUD against in-memory SQLite
                # (repositories/beego/task_repository_crud_test.go)
pola serve      # then: curl localhost:3000/tasks
```
