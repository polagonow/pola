table "products" {
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
  column "price" {
    null = true
    type = real
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_products_deleted_at" {
    columns = [column.deleted_at]
  }
}
schema "main" {
}
