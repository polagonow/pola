export type Metadata = {
  title?: string | { default?: string; template?: string; absolute?: string }
  description?: string
  applicationName?: string
  authors?: Array<{ name?: string; url?: string }>
  generator?: string
  keywords?: string | string[]
  referrer?: string
  creator?: string
  publisher?: string

  robots?: {
    index?: boolean
    follow?: boolean
    noarchive?: boolean
    nosnippet?: boolean
    noimageindex?: boolean
    nocache?: boolean
    notranslate?: boolean
    maxSnippet?: number
    maxImagePreview?: 'none' | 'standard' | 'large'
    maxVideoPreview?: number
  }
  alternates?: {
    canonical?: string
    languages?: Record<string, string>
  }
  verification?: {
    google?: string
    yandex?: string
    yahoo?: string
    me?: string
  }

  icons?: {
    icon?: Array<{ url: string; type?: string; sizes?: string }> | string
    shortcut?: Array<{ url: string; sizes?: string }> | string
    apple?: Array<{ url: string; sizes?: string; type?: string }> | string
    other?: Array<{ rel: string; url: string; type?: string; sizes?: string }>
  }

  openGraph?: {
    type?: string
    title?: string
    description?: string
    url?: string
    siteName?: string
    locale?: string
    images?: string[] | Array<{ url: string; secureUrl?: string; type?: string; alt?: string; width?: number; height?: number }>
  }
  twitter?: {
    card?: 'summary' | 'summary_large_image' | 'app' | 'player'
    site?: string
    creator?: string
    title?: string
    description?: string
    images?: string[]
  }

  other?: Record<string, string>
}

export type GenerateMetadata<
  TParams extends Record<string, string | string[]> = Record<string, string>,
  TSearchParams extends Record<string, string | string[] | undefined> = Record<string, string | string[] | undefined>,
> = (props: {
  params: TParams
  searchParams: TSearchParams
}) => Metadata | Promise<Metadata>
