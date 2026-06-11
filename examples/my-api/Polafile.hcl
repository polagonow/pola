pola {
  package      = "github.com/owner/my-api"
  version      = "272dd15-dirty"
  api_only     = true
  routes       = "routes"
  repositories = "repositories"
  services     = "services"

  csrf {
    enabled = true
  }

  security_headers {
    enabled = true
  }

  cache {
    enabled = true
    adapter = "memory"
  }
}
