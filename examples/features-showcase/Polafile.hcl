pola {
  package      = "features-showcase"
  version      = "dev"
  api_only     = true
  routes       = "routes"
  repositories = "repositories"
  services     = "services"

  csrf {
  }

  security_headers {
  }

  cache {
    adapter = "memory"
  }

  database {
    adapter = "sqlite"
    orm     = "gorm"
  }
}
