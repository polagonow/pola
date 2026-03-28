// Global 404 — rendered when no route matches at all.
// Must define its own <html>/<body> since it bypasses all layouts.
export default function GlobalNotFound() {
  return (
    <html lang="en">
      <head>
        <meta charSet="UTF-8" />
        <meta name="viewport" content="width=device-width,initial-scale=1" />
        <title>404 — DevBlog</title>
        <style>{`
          body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;
               min-height:100vh;margin:0;background:#f9fafb;color:#111}
          .box{text-align:center;padding:2rem}
          .code{font-size:5rem;font-weight:800;color:#e5e7eb;line-height:1}
          h1{font-size:1.4rem;font-weight:600;margin:.75rem 0 .5rem}
          p{color:#6b7280;margin-bottom:1.5rem}
          a{display:inline-block;padding:.5rem 1.25rem;background:#2563eb;color:#fff;
            border-radius:8px;text-decoration:none;font-weight:500}
        `}</style>
      </head>
      <body>
        <div className="box">
          <div className="code">404</div>
          <h1>Page not found</h1>
          <p>That URL doesn't match any route.</p>
          <a href="/">Go home</a>
        </div>
      </body>
    </html>
  );
}
