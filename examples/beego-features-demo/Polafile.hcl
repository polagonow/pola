pola {
  package         = "beego-features-demo"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
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
    enabled = false
  }

  database {
    models  = "db/models"
    orm     = "gorm"
    adapter = "sqlite"

    migrations {
      directory = "db/migrations"
      format    = "sql"
    }
  }

  session {
    enabled = true
    store   = "memory"
    max_age = "24h"
  }

  rate_limit {
    enabled             = true
    requests_per_second  = 10
    burst               = 20
  }

  flash {
    enabled = true
  }

  i18n {
    enabled        = true
    default_locale = "en"
    directory      = "locales"
  }

}
