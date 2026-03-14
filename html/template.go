package html

// shellHead is the HTML document head, closed at </head><body>.
// The import map and body content are appended after it.
const shellHead = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>GoJSX</title>
  <style>
    *,*::before,*::after{box-sizing:border-box}
    body{font-family:system-ui,sans-serif;margin:0;padding:2rem;background:#f8f8f8;color:#111}
    .page{max-width:860px;margin:0 auto}
    h1{font-size:2rem;margin:0 0 .5rem}
    h2{font-size:1.2rem;margin:1.5rem 0 .5rem;color:#444}
    code{background:#eee;padding:.1em .35em;border-radius:4px;font-size:.9em}
    nav{display:flex;gap:1rem;margin-bottom:1.5rem;padding-bottom:1rem;border-bottom:1px solid #e0e0e0}
    nav a{color:#0070f3;text-decoration:none;font-weight:500}
    nav a:hover{text-decoration:underline}
    .product-list{display:grid;gap:.5rem;margin-top:.5rem}
    .product{background:#fff;padding:.75rem 1rem;border-radius:8px;border:1px solid #e5e5e5;
             display:flex;align-items:center;gap:.75rem}
    .product strong{flex:1}
    .product .price{color:#0070f3;font-weight:600}
    .product small{color:#999}
    dl{display:grid;grid-template-columns:max-content 1fr;gap:.25rem 1.5rem;margin-top:.5rem}
    dt{font-weight:600;color:#555}
    #root{padding-top:.5rem}
    .rsc-err{color:#c0392b;background:#fdecea;padding:.75rem 1rem;border-radius:8px}
  </style>
</head>
<body>
`

// shellBody is the fixed body content placed after the import map.
const shellBody = `
  <nav>
    <a href="/">Home</a>
    <a href="/products">Products</a>
    <a href="/user?id=42">Profile</a>
    <a href="/about">About</a>
  </nav>
  <div id="root"></div>
`
