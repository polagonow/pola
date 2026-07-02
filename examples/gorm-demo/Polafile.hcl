pola {
  package      = "gorm-demo"
  version      = "dev"
  api_only     = true
  routes       = "routes"
  repositories = "repositories"
  services     = "services"

  csrf {
  }

  security_headers {
  }

  database {
    orm     = "gorm"
    adapter = "sqlite"
  }

  cache {
    adapter = "memory"
  }
}
