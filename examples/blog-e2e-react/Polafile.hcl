pola {
  package         = "blog-e2e-react"
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
    models  = "db/models"
    orm     = "ent"

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
