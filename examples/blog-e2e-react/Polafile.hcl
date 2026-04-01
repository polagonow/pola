pola {
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  cache           = "memory"
  package_manager = "pnpm"
  csrf            = "true"
  security_headers = "true"
  app_dir         = "app"
  actions_dir     = "actions"
  routes_dir      = "routes"
}
