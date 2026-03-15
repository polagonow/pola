// The global-not-found.tsx file lets you define a 404 page for your entire application. Unlike not-found.js, which works at the route level, this is used when a requested URL doesn't match any route at all. Next.js skips rendering and directly returns this global page.

// The global-not-found.tsx file bypasses your app's normal rendering, which means you'll need to import any global styles, fonts, or other dependencies that your 404 page requires.
export default function GlobalNotFound() {
  return (
    <html lang="en">
      <body>
        <h1>404 - Page Not Found</h1>
        <p>This page does not exist.</p>
      </body>
    </html>
  )
}
