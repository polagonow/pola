pola {
  package         = "slds-test"
  version         = "469a8b1-dirty"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "none"
  ui              = "slds"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"
  repositories    = "repositories"
  services        = "services"

  csrf {}

  security_headers {}

  cache {
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
