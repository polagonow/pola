pola {
  package         = "fumadocs-docs"
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
    adapter = "memory"
  }

  testing {
    generate_tests = false
    framework      = "vitest"
  }
}
