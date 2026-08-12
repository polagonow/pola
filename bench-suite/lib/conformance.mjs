// DOM conformance normalization.
//
// The gate asserts that every implementation of a scenario produces equivalent
// *rendered* DOM. We normalize by:
//   - removing <script>, <style>, <template>, and HTML comments
//   - removing framework-specific attributes (data-reactroot, hidden markers,
//     RSC/hydration ids, nonce, class ordering is preserved but framework hint
//     attrs are dropped)
//   - collapsing whitespace between tags and trimming text nodes
//
// For SSR entries the input is the first-response HTML. For RSC entries whose
// first response is a shell (e.g. Pola), the input MUST be the browser-rendered
// DOM (captured via Playwright after network-idle) — passing raw shell HTML
// here would normalize to near-empty and (correctly) fail conformance, which is
// why the orchestrator marks such entries as needing browser capture.

const DROP_TAGS = ["script", "style", "template", "noscript"];

// Attributes that are framework bookkeeping, not user-visible content.
const DROP_ATTR_RE =
  /\s(?:data-reactroot|data-react-[a-z-]+|data-rsc[a-z-]*|data-n-[a-z-]+|data-pola[a-z-]*|nonce|data-precedence|data-nscript|data-turbo[a-z-]*|hidden(?=\s|>|=))(?:="[^"]*")?/gi;

export function normalizeDom(html, { rootSelectorHint } = {}) {
  let s = String(html);

  // Drop the document shell wrapper differences: keep only body inner if present.
  const bodyMatch = s.match(/<body[^>]*>([\s\S]*?)<\/body>/i);
  if (bodyMatch) s = bodyMatch[1];

  // Remove HTML comments (RSC/hydration boundaries like <!--$--> <!--/$-->).
  s = s.replace(/<!--[\s\S]*?-->/g, "");

  // Remove dropped tags and their content.
  for (const tag of DROP_TAGS) {
    const re = new RegExp(`<${tag}\\b[^>]*>[\\s\\S]*?<\\/${tag}>`, "gi");
    s = s.replace(re, "");
    // self-closing / void form
    s = s.replace(new RegExp(`<${tag}\\b[^>]*/?>`, "gi"), "");
  }

  // Remove the empty RSC mount root wrappers that carry no content.
  s = s.replace(DROP_ATTR_RE, "");

  // Normalize whitespace.
  s = s
    .replace(/>\s+</g, "><")
    .replace(/\s{2,}/g, " ")
    .replace(/^\s+|\s+$/g, "");

  return s;
}

// Cheap structural signature: sequence of tag names + text tokens. Two DOMs that
// normalize to the same signature are considered conformant for the gate.
export function domSignature(html, opts) {
  const norm = normalizeDom(html, opts);
  const tokens = [];
  const re = /<\/?([a-zA-Z][a-zA-Z0-9-]*)\b[^>]*>|([^<]+)/g;
  let m;
  while ((m = re.exec(norm))) {
    if (m[1]) tokens.push(m[1].toLowerCase());
    else if (m[2]) {
      const t = m[2].replace(/\s+/g, " ").trim();
      if (t) tokens.push(`#${t}`);
    }
  }
  return tokens.join("|");
}

// Compare a map of { entryName: html } for one scenario. Returns
// { conformant: bool, groups: { signature: [entryNames] }, reference }.
export function compareScenario(entryHtml) {
  const groups = {};
  for (const [name, html] of Object.entries(entryHtml)) {
    if (html == null) continue;
    const sig = domSignature(html);
    (groups[sig] ??= []).push(name);
  }
  const sigs = Object.keys(groups);
  return {
    conformant: sigs.length <= 1,
    distinctGroups: sigs.length,
    groups,
  };
}
