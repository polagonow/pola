pola {
  package         = "image-demo"
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
    enabled = true
    adapter = "memory"
  }

  # Enables the on-the-fly image optimizer served at /_pola/image, which the
  # @pola/react/image <Image> component routes through. The "imaging" adapter is
  # pure Go (no CGO). Output defaults to JPEG; resize/crop/quality are driven by
  # the query string the component generates.
  image_processing {
    enabled    = true
    adapter    = "imaging"
    path       = "/_pola/image"
    max_width  = 4096
    max_height = 4096
    format     = "jpeg"
  }

  testing {
    generate_tests = false
    framework      = "vitest"
  }
}
