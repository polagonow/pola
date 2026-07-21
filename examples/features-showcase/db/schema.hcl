table "products" {
  schema = schema.main
  column "id" {
    null           = true
    type           = integer
    auto_increment = true
  }
  column "name" {
    null = true
    type = varchar
  }
  column "price" {
    null = true
    type = real
  }
  column "description" {
    null = true
    type = text
  }
  column "created_at" {
    null = true
    type = datetime
  }
  column "updated_at" {
    null = true
    type = datetime
  }
  primary_key {
    columns = [column.id]
  }
}
schema "main" {
}
