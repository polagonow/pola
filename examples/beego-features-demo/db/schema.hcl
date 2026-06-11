table "users" {
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
  column "username" {
    null = true
    type = varchar
  }
  column "password_hash" {
    null = true
    type = varchar
  }
  column "display_name" {
    null = true
    type = varchar
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_users_username" {
    unique  = true
    columns = [column.username]
  }
  index "idx_users_deleted_at" {
    columns = [column.deleted_at]
  }
}
schema "main" {
}
