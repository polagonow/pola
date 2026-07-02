table "tasks" {
  schema = schema.main
  column "id" {
    null           = false
    type           = integer
    auto_increment = true
  }
  column "title" {
    null    = false
    type    = varchar
    default = ""
  }
  column "done" {
    null    = false
    type    = bool
    default = false
  }
  primary_key {
    columns = [column.id]
  }
}
schema "main" {
}
