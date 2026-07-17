pola {
  package         = "saas-starter"
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
    # Stripe's webhook authenticates via its own signature and cannot present a
    # CSRF token, so exempt it from CSRF protection.
    exempt = ["/api/stripe/webhook"]
  }

  security_headers {
  }

  cache {
    adapter = "memory"
  }

  # JWT-cookie sessions (the pola-native equivalent of the starter's jose JWT):
  # HS256, keyed off AUTH_SECRET, stored in the "session" cookie, 1-day expiry.
  jwt {
    cookie     = "session"
    expiry     = "24h"
    secret_env = "AUTH_SECRET"
  }

  database {
    orm     = "gorm"
    adapter = "sqlite"
    models  = "db/models"

    migrations {
      directory = "db/migrations"
      format    = "sql"
    }

    env "production" {
      adapter = "postgresql"
    }
  }

  testing {
    generate_tests = true
    framework      = "vitest"
  }
}
