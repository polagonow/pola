// Build the control entry with esbuild:
//   - dist/server/index.mjs   server render (node, react kept external)
//   - dist/client/s3.js       minified browser hydration bundle for scenario 3
//   - dist/client/report.json framework-vs-app RAW byte attribution (from the
//                             esbuild metafile — exact, post-minify/treeshake)

import * as esbuild from "esbuild";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dir = path.dirname(fileURLToPath(import.meta.url));
const p = (...s) => path.join(dir, ...s);

fs.mkdirSync(p("dist/server"), { recursive: true });
fs.mkdirSync(p("dist/client"), { recursive: true });

// Server bundle — keep node_modules (react, react-dom) external; resolved at
// runtime. Production React is selected at runtime via NODE_ENV=production
// (set by the server's start env), so no define is needed here.
await esbuild.build({
  entryPoints: [p("src/server-render.jsx")],
  bundle: true,
  platform: "node",
  format: "esm",
  packages: "external",
  jsx: "automatic",
  outfile: p("dist/server/index.mjs"),
  logLevel: "info",
});

// Client bundle — the actual JS shipped to the browser for scenario 3.
const result = await esbuild.build({
  entryPoints: { s3: p("src/entry-client-s3.jsx") },
  bundle: true,
  platform: "browser",
  format: "esm",
  minify: true,
  jsx: "automatic",
  define: { "process.env.NODE_ENV": '"production"' }, // production React build
  outdir: p("dist/client"),
  entryNames: "[name]",
  metafile: true,
  logLevel: "info",
});

fs.writeFileSync(p("dist/client/meta.json"), JSON.stringify(result.metafile, null, 2));

// Attribute output bytes to framework (react/react-dom/scheduler) vs app code.
let frameworkRawBytes = 0;
let appRawBytes = 0;
const FRAMEWORK_RE = /node_modules\/(react|react-dom|scheduler)(\/|$)/;
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
      note: "RAW output bytes attributed via esbuild metafile (post-minify, post-treeshake). gzip/brotli of the whole client bundle are reported separately by the harness.",
      frameworkRawBytes,
      appRawBytes,
      totalRawBytes: frameworkRawBytes + appRawBytes,
    },
    null,
    2,
  ),
);

console.log(
  `client bundle: framework=${frameworkRawBytes}B app=${appRawBytes}B total=${frameworkRawBytes + appRawBytes}B`,
);
