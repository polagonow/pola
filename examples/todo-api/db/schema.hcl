table "todos" {
  schema = schema.main
  column "id" {
    null           = true
    type           = integer
    auto_increment = true
  }
  column "created_at" {
    null = true
    type = datetime
  }
  column "updated_at" {
    null = true
    type = datetime
  }
  column "deleted_at" {
    null = true
    type = datetime
  }
  column "title" {
    null = true
    type = varchar
  }
  column "completed" {
    null = true
    type = numeric
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_todos_deleted_at" {
    columns = [column.deleted_at]
  }
}
schema "main" {
}
