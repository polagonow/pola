pola {
  package      = "todo-api"
  api_only     = true
  routes       = "routes"
  repositories = "repositories"
  services     = "services"

  database {
    url     = env("DATABASE_URL")
    adapter = "postgresql"
    orm     = "gorm"
  }

  csrf {
    enabled = false
  }

  security_headers {
    enabled = true
  }

  cache {
    enabled = true
    adapter = "memory"
  }
}
