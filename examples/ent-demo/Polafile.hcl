pola {
  package      = "ent-demo"
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
    orm     = "ent"
    adapter = "sqlite"
  }

  cache {
    adapter = "memory"
  }
}
