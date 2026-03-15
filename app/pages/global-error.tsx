// Global errors
// While less common, you can handle errors in the root layout using the global-error.js file, located in the root app directory, even when leveraging internationalization. Global error UI must define its own <html> and <body> tags, since it is replacing the root layout or template when active.

'use client' // Error boundaries must be Client Components

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  return (
    // global-error must include html and body tags
    <html>
      <body>
        <h2>Something went wrong!</h2>
        <button onClick={() => reset()}>Try again</button>
      </body>
    </html>
  )
}
