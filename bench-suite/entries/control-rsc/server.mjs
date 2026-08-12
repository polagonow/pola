// RSC baseline HTTP server. MUST run under `--conditions react-server` (see the
// start script) so `react` resolves to the react-server build and Server
// Components can run. Implements the same two-request model as Pola: a normal
// GET returns an HTML shell; `Accept: text/x-component` streams the Flight payload.

import http from "node:http";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { renderToPipeableStream } from "react-server-dom-webpack/server.node";
import { scenarios } from "./dist/server/scenarios.mjs";

const dir = path.dirname(fileURLToPath(import.meta.url));
const PORT = parseInt(process.env.PORT || "3000", 10);
const CLIENT_DIR = path.join(dir, "dist/client");
const FLIGHT = "text/x-component";

// No client components in these scenarios, so the client-reference module map is empty.
const MODULE_MAP = {};

function shell() {
  return (
    "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\">" +
    "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">" +
    "<title>RSC baseline</title></head><body><div id=\"root\"></div>" +
    "<script type=\"module\" src=\"/client/main.js\"></script></body></html>"
  );
}

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

  const m = pathname.match(/^\/s([124])$/);
  if (m) {
    const factory = scenarios[m[1]];
    if (!factory) {
      res.statusCode = 404;
      res.end("scenario not available");
      return;
    }
    const isFlight =
      req.headers["content-type"] === FLIGHT || req.headers["accept"] === FLIGHT;
    if (isFlight) {
      res.statusCode = 200;
      res.setHeader("content-type", FLIGHT + "; charset=utf-8");
      res.setHeader("cache-control", "no-store");
      const { pipe } = renderToPipeableStream(factory(), MODULE_MAP, {
        onError(err) {
          process.stderr.write("rsc render error: " + String(err) + "\n");
        },
      });
      pipe(res);
    } else {
      res.statusCode = 200;
      res.setHeader("content-type", "text/html; charset=utf-8");
      res.end(shell());
    }
    return;
  }

  res.statusCode = 404;
  res.end("not found");
});

server.listen(PORT, "127.0.0.1", () => {
  process.stdout.write(`control-rsc listening on http://127.0.0.1:${PORT}\n`);
});
