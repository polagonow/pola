table "users" {
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
  column "email" {
    null = true
    type = varchar
  }
  column "password_hash" {
    null = true
    type = varchar
  }
  column "role" {
    null = true
    type = varchar
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
  primary_key {
    columns = [column.id]
  }
  index "idx_users_email" {
    unique  = true
    columns = [column.email]
  }
}
table "teams" {
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
  column "stripe_customer_id" {
    null = true
    type = varchar
  }
  column "stripe_subscription_id" {
    null = true
    type = varchar
  }
  column "stripe_product_id" {
    null = true
    type = varchar
  }
  column "plan_name" {
    null = true
    type = varchar
  }
  column "subscription_status" {
    null = true
    type = varchar
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
  index "idx_teams_stripe_subscription_id" {
    unique  = true
    columns = [column.stripe_subscription_id]
  }
  index "idx_teams_stripe_customer_id" {
    unique  = true
    columns = [column.stripe_customer_id]
  }
}
table "team_members" {
  schema = schema.main
  column "id" {
    null           = true
    type           = integer
    auto_increment = true
  }
  column "user_id" {
    null = true
    type = integer
  }
  column "team_id" {
    null = true
    type = integer
  }
  column "role" {
    null = true
    type = varchar
  }
  column "joined_at" {
    null = true
    type = datetime
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
table "activity_logs" {
  schema = schema.main
  column "id" {
    null           = true
    type           = integer
    auto_increment = true
  }
  column "team_id" {
    null = true
    type = integer
  }
  column "user_id" {
    null = true
    type = integer
  }
  column "action" {
    null = true
    type = varchar
  }
  column "timestamp" {
    null = true
    type = datetime
  }
  column "ip_address" {
    null = true
    type = varchar
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
table "invitations" {
  schema = schema.main
  column "id" {
    null           = true
    type           = integer
    auto_increment = true
  }
  column "team_id" {
    null = true
    type = integer
  }
  column "email" {
    null = true
    type = varchar
  }
  column "invited_by_id" {
    null = true
    type = integer
  }
  column "role" {
    null = true
    type = varchar
  }
  column "invited_at" {
    null = true
    type = datetime
  }
  column "status" {
    null = true
    type = varchar
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
