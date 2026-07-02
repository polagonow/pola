table "tasks" {
  schema = schema.main
  column "id" {
    null           = false
    type           = integer
    auto_increment = true
  }
  column "title" {
    null = false
    type = text
  }
  column "done" {
    null = false
    type = bool
  }
  primary_key {
    columns = [column.id]
  }
}
schema "main" {
}
