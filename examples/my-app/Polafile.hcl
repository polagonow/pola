pola {
  package         = "my-app"
  version         = "dev"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  ui              = "shadcn"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"
  repositories    = "repositories"
  services        = "services"

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

  database {
    models  = "db/models"
    adapter = "sqlite"
    orm     = "gorm"
    migrations {
      directory = "db/migrations"
      format    = "hcl"
    }
    env "development" {
      adapter = "sqlite"
    }
    env "production" {
      adapter = "postgresql"
    }
  }
}
