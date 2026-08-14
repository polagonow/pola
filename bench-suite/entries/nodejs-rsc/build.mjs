// Build the Node.js RSC baseline with esbuild:
//   - dist/server/scenarios.mjs  server-component trees (node; react kept
//                                external so the runtime resolves the
//                                `react-server` condition build)
//   - dist/client/main.js        browser Flight client (createFromFetch)
//   - dist/client/report.json    framework-vs-app RAW byte attribution

import * as esbuild from "esbuild";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dir = path.dirname(fileURLToPath(import.meta.url));
const p = (...s) => path.join(dir, ...s);

fs.mkdirSync(p("dist/server"), { recursive: true });
fs.mkdirSync(p("dist/client"), { recursive: true });

// Server bundle — react/react-server-dom-webpack stay external; resolved at
// runtime under `--conditions react-server` (set by the start command).
await esbuild.build({
  entryPoints: [p("src/scenarios.server.jsx")],
  bundle: true,
  platform: "node",
  format: "esm",
  packages: "external",
  jsx: "automatic",
  outfile: p("dist/server/scenarios.mjs"),
  logLevel: "info",
});

// Client bundle — the Flight client shipped to the browser.
const result = await esbuild.build({
  entryPoints: { main: p("src/entry-client.jsx") },
  bundle: true,
  platform: "browser",
  format: "esm",
  minify: true,
  jsx: "automatic",
  define: { "process.env.NODE_ENV": '"production"' },
  conditions: ["browser", "module", "default"], // NOT react-server (that's server-only)
  outdir: p("dist/client"),
  entryNames: "[name]",
  metafile: true,
  logLevel: "info",
});

fs.writeFileSync(p("dist/client/meta.json"), JSON.stringify(result.metafile, null, 2));

let frameworkRawBytes = 0;
let appRawBytes = 0;
const FRAMEWORK_RE = /node_modules\/(react|react-dom|react-server-dom-webpack|scheduler)(\/|$)/;
for (const outInfo of Object.values(result.metafile.outputs)) {
  for (const [input, det] of Object.entries(outInfo.inputs)) {
    if (FRAMEWORK_RE.test(input)) frameworkRawBytes += det.bytesInOutput;
    else appRawBytes += det.bytesInOutput;
  }
}
fs.writeFileSync(
  p("dist/client/report.json"),
  JSON.stringify(
    {
      note: "RAW output bytes attributed via esbuild metafile (post-minify/treeshake).",
      frameworkRawBytes,
      appRawBytes,
      totalRawBytes: frameworkRawBytes + appRawBytes,
    },
    null,
    2,
  ),
);

console.log(`client bundle: framework=${frameworkRawBytes}B app=${appRawBytes}B`);
