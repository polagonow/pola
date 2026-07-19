"use client";

// Client re-exports of the fumadocs-ui block components used by Pola's MDX
// pipeline. Marking them "use client" turns them into RSC client references, so
// their implementation (and heavy transitive deps like lucide-react, whose
// icon-name regex the pure-Go goja engine's RE2 cannot compile) is bundled for
// the browser only and never loaded into the server JS engine. The components are
// purely presentational, so client rendering is equivalent — their
// server-rendered children (the compiled Markdown) pass straight through.

export { Callout } from "fumadocs-ui/components/callout";
export { Card, Cards } from "fumadocs-ui/components/card";
export { CodeBlock, Pre } from "fumadocs-ui/components/codeblock";
