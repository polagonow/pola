// Payload and client-bundle size helpers.

import zlib from "node:zlib";
import fs from "node:fs";
import path from "node:path";

export function gzipBytes(buf) {
  return zlib.gzipSync(buf, { level: zlib.constants.Z_BEST_COMPRESSION }).length;
}

export function brotliBytes(buf) {
  return zlib.brotliCompressSync(buf, {
    params: { [zlib.constants.BROTLI_PARAM_QUALITY]: 11 },
  }).length;
}

// { raw, gzip, brotli } for a Buffer/string.
export function sizeReport(buf) {
  const b = Buffer.isBuffer(buf) ? buf : Buffer.from(String(buf));
  return { raw: b.length, gzip: gzipBytes(b), brotli: brotliBytes(b) };
}

// Sum sizes of files matching the given absolute-or-relative glob-ish patterns.
// Patterns support a leading dir + '*' wildcard on the basename only (enough for
// bundler outputs like `dist/vendor-*.js`). Returns { raw, gzip, brotli, files }.
export function bundleSize(baseDir, patterns) {
  const matched = [];
  for (const pat of patterns ?? []) {
    const dir = path.resolve(baseDir, path.dirname(pat));
    const base = path.basename(pat);
    if (!fs.existsSync(dir)) continue;
    const re = globToRegExp(base);
    for (const name of fs.readdirSync(dir)) {
      if (re.test(name)) matched.push(path.join(dir, name));
    }
  }
  let raw = 0;
  let gzip = 0;
  let brotli = 0;
  const files = [];
  for (const f of [...new Set(matched)]) {
    const buf = fs.readFileSync(f);
    raw += buf.length;
    gzip += gzipBytes(buf);
    brotli += brotliBytes(buf);
    files.push(path.relative(baseDir, f));
  }
  return { raw, gzip, brotli, files: files.sort() };
}

function globToRegExp(glob) {
  const escaped = glob.replace(/[.+^${}()|[\]\\]/g, "\\$&").replace(/\*/g, ".*");
  return new RegExp(`^${escaped}$`);
}
