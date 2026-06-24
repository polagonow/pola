table "todos" {
  schema = schema.main
  column "id" {
    null = true
    type = sql("bigserial")
  }
  column "created_at" {
    null = true
    type = sql("timestamptz")
  }
  column "updated_at" {
    null = true
    type = sql("timestamptz")
  }
  column "deleted_at" {
    null = true
    type = sql("timestamptz")
  }
  column "title" {
    null = true
    type = varchar
  }
  column "completed" {
    null = true
    type = boolean
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
