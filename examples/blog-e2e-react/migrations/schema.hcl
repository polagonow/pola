table "texts" {
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
  column "name" {
    null = true
    type = varchar
  }
  column "password" {
    null = true
    type = varchar
  }
  column "age" {
    null = true
    type = integer
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_texts_deleted_at" {
    columns = [column.deleted_at]
  }
}
schema "main" {
}
