pola {
  package      = "beego-demo"
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
    orm     = "beego"
    adapter = "sqlite"
  }

  cache {
    adapter = "memory"
  }
}
