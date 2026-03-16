package shell

// shellTemplate is the full HTML document template.
const shellTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>DevBlog</title>
  <link rel="icon" href="/public/favicon.ico">
  <style>
    :root{
      --bg:#fff;--fg:#111;--muted:#6b7280;--border:#e5e7eb;
      --accent:#2563eb;--accent-fg:#fff;--surface:#f9fafb;
      --radius:8px;--shadow:0 1px 3px rgba(0,0,0,.08);
    }
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    body{font-family:system-ui,sans-serif;background:var(--bg);color:var(--fg);line-height:1.6}
    a{color:var(--accent);text-decoration:none}
    a:hover{text-decoration:underline}
    code{background:var(--surface);border:1px solid var(--border);padding:.1em .4em;border-radius:4px;font-size:.875em}

    /* ── Shell ── */
    .shell{min-height:100vh;display:flex;flex-direction:column}
    .topnav{border-bottom:1px solid var(--border);padding:.75rem 1.5rem;display:flex;align-items:center;gap:2rem}
    .topnav .brand{font-weight:700;font-size:1.1rem;color:var(--fg)}
    .topnav nav{display:flex;gap:1.5rem}
    .topnav nav a{color:var(--muted);font-size:.9rem;font-weight:500}
    .topnav nav a:hover{color:var(--fg);text-decoration:none}
    .topnav nav a.nav-active{color:var(--fg);font-weight:600}
    .topnav .actions{margin-left:auto;display:flex;align-items:center;gap:.75rem}
    .main{flex:1;max-width:1000px;margin:0 auto;width:100%;padding:2rem 1.5rem}

    /* ── Cards ── */
    .card{background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);
          padding:1.25rem 1.5rem;transition:box-shadow .15s}
    .card:hover{box-shadow:var(--shadow)}
    .card-grid{display:grid;gap:1rem}
    .card h2{font-size:1.1rem;font-weight:600;margin-bottom:.25rem}
    .card h3{font-size:1rem;font-weight:600;margin-bottom:.25rem}
    .card p{color:var(--muted);font-size:.9rem;margin-bottom:.5rem}

    /* ── Tags ── */
    .tags{display:flex;flex-wrap:wrap;gap:.35rem;margin-top:.5rem}
    .tag{background:var(--surface);border:1px solid var(--border);border-radius:99px;
         padding:.15em .65em;font-size:.78rem;color:var(--muted)}

    /* ── Meta ── */
    .meta{font-size:.82rem;color:var(--muted);display:flex;flex-wrap:wrap;gap:.75rem;margin-bottom:.5rem}

    /* ── Hero ── */
    .hero{padding:3rem 0 2rem}
    .hero h1{font-size:2.5rem;font-weight:800;letter-spacing:-.02em;line-height:1.15}
    .hero p{color:var(--muted);font-size:1.1rem;margin-top:.75rem;max-width:520px}
    .hero-actions{display:flex;gap:.75rem;margin-top:1.5rem}

    /* ── Buttons ── */
    .btn{display:inline-flex;align-items:center;gap:.4rem;padding:.5rem 1rem;border-radius:var(--radius);
         font-size:.9rem;font-weight:500;cursor:pointer;border:1px solid transparent;transition:all .15s}
    .btn-primary{background:var(--accent);color:var(--accent-fg);border-color:var(--accent)}
    .btn-primary:hover{opacity:.9;text-decoration:none}
    .btn-outline{background:transparent;color:var(--fg);border-color:var(--border)}
    .btn-outline:hover{background:var(--surface);text-decoration:none}
    .btn-ghost{background:transparent;color:var(--muted);border-color:transparent;padding:.35rem .6rem}
    .btn-ghost:hover{background:var(--surface);color:var(--fg);text-decoration:none}

    /* ── Sidebar layout (route groups) ── */
    .sidebar-layout{display:grid;grid-template-columns:200px 1fr;gap:2rem;align-items:start}
    .sidebar{position:sticky;top:1.5rem}
    .sidebar h4{font-size:.78rem;font-weight:600;text-transform:uppercase;
                letter-spacing:.05em;color:var(--muted);margin-bottom:.75rem}
    .sidebar ul{list-style:none;display:flex;flex-direction:column;gap:.25rem}
    .sidebar ul li a{font-size:.88rem;color:var(--muted);padding:.2rem 0;display:block}
    .sidebar ul li a:hover{color:var(--fg);text-decoration:none}

    /* ── Section badge (route group header) ── */
    .section-badge{display:inline-flex;align-items:center;gap:.4rem;font-size:.78rem;
                   font-weight:600;text-transform:uppercase;letter-spacing:.06em;
                   color:var(--muted);margin-bottom:1.25rem}

    /* ── Detail page ── */
    .detail-header{margin-bottom:1.5rem}
    .detail-header h1{font-size:1.8rem;font-weight:800;line-height:1.2}
    .back-link{font-size:.85rem;color:var(--muted);margin-bottom:1rem;display:inline-block}
    .back-link:hover{color:var(--fg)}

    /* ── Profile ── */
    .profile-card{display:flex;gap:1.5rem;align-items:flex-start}
    .avatar{width:72px;height:72px;border-radius:50%;background:var(--accent);
            display:flex;align-items:center;justify-content:center;
            font-size:1.8rem;font-weight:700;color:#fff;flex-shrink:0}
    .profile-info h1{font-size:1.4rem;font-weight:700}
    .profile-info .role{color:var(--muted);font-size:.9rem;margin-bottom:.5rem}

    /* ── Status badge ── */
    .status{display:inline-block;padding:.1em .55em;border-radius:99px;font-size:.75rem;font-weight:600}
    .status-active{background:#dcfce7;color:#166534}
    .status-stable{background:#dbeafe;color:#1e40af}
    .status-beta{background:#fef9c3;color:#854d0e}

    /* ── Tech list ── */
    .tech-list{display:flex;flex-wrap:wrap;gap:.4rem;margin-top:.5rem}
    .tech{background:var(--surface);border:1px solid var(--border);border-radius:4px;
          padding:.1em .5em;font-size:.8rem;font-family:monospace}

    /* ── Misc ── */
    .section-title{font-size:1.2rem;font-weight:700;margin-bottom:1rem}
    .grid-2{display:grid;grid-template-columns:1fr 1fr;gap:1rem}
    .rsc-err{color:#dc2626;background:#fef2f2;padding:.75rem 1rem;border-radius:var(--radius);border:1px solid #fecaca}
    @keyframes pulse{0%,100%{opacity:1}50%{opacity:.45}}
    #root{padding-top:.25rem}
    @media(max-width:640px){
      .sidebar-layout{grid-template-columns:1fr}
      .sidebar{display:none}
      .grid-2{grid-template-columns:1fr}
      .hero h1{font-size:1.8rem}
    }
  </style>
  {{.ImportMap}}
</head>
<body>
  <div id="root"></div>
{{range .Scripts}}  <script>{{.}}</script>
{{end}}  <script type="module" src="{{.ClientScript}}"></script>
</body>
</html>`
