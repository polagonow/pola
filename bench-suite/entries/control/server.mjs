// Control HTTP server: plain Node http + react-dom/server streaming.
// Listens on PORT (default 3000). No compression, no caching — the raw floor.

import http from "node:http";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { renderScenario } from "./dist/server/index.mjs";

const dir = path.dirname(fileURLToPath(import.meta.url));
const PORT = parseInt(process.env.PORT || "3000", 10);
const CLIENT_DIR = path.join(dir, "dist/client");

const server = http.createServer((req, res) => {
  const url = new URL(req.url, "http://localhost");
  const pathname = url.pathname;

  if (pathname === "/health") {
    res.statusCode = 200;
    res.end("ok");
    return;
  }

  if (pathname.startsWith("/client/")) {
    const file = path.join(CLIENT_DIR, pathname.slice("/client/".length));
    if (file.startsWith(CLIENT_DIR) && fs.existsSync(file)) {
      res.statusCode = 200;
      res.setHeader("content-type", "text/javascript; charset=utf-8");
      fs.createReadStream(file).pipe(res);
      return;
    }
    res.statusCode = 404;
    res.end("not found");
    return;
  }

  const m = pathname.match(/^\/s([1-4])$/);
  if (m) {
    renderScenario(m[1], res);
    return;
  }

  res.statusCode = 404;
  res.end("not found");
});

server.listen(PORT, "127.0.0.1", () => {
  process.stdout.write(`control listening on http://127.0.0.1:${PORT}\n`);
});
