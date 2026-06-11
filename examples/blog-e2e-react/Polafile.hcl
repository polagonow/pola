pola {
  package         = "blog-e2e-react"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"

  csrf {}

  security_headers {}

  cache {
    adapter = "memory"
  }

  database {
    models  = "db/models"
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
