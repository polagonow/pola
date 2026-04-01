pola {
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  package_manager = "pnpm"
  app             = "app"
  actions         = "actions"
  routes          = "routes"

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
    models     = "models"
    migrations = "migrations"
    orm        = "ent"

    environment "development" {
      adapter = "sqlite"
    }

    environment "production" {
      adapter = "postgresql"
    }
  }
}
