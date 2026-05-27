pola {
  package         = "mcp-hello"
  version         = "dev"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  ui              = "none"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"
  repositories    = "repositories"
  services        = "services"

  csrf {
    enabled = false
  }

  security_headers {
    enabled = false
  }

  cache {
    enabled = true
    adapter = "memory"
  }

  database {
    models  = "db/models"
    orm     = "gorm"
    adapter = "sqlite"

    migrations {
      directory = "db/migrations"
      format    = "hcl"
    }
  }

  mcp {
    enabled   = true
    transport = "http"
    mount     = "/mcp"
  }
}
