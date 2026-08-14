pola {
  package         = "app"
  version         = "v0.1.0-6-gd808cc0"
  renderer        = "react"
  engine          = "sobek"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "none"
  ui              = "none"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"
  repositories    = "repositories"
  services        = "services"

  csrf {
  }

  security_headers {
  }

  cache {
    adapter = "memory"
  }

  testing {
    generate_tests = true
    framework      = "vitest"
  }
}
