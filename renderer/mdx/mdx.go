// Package mdx provides a Node-free Markdown/MDX compiler that transpiles content
// files into React modules compatible with fumadocs-core's source loader and
// fumadocs-ui. It is built on goldmark (CommonMark + GFM) and runs entirely
// inside pola's Go build pipeline — there is no JavaScript/Node toolchain, so a
// docs site compiles as part of the normal single-binary build.
//
// The generated module exports the four values fumadocs' `.source` contract
// expects, so `fumadocs-core/source`'s loader() and fumadocs-ui can consume it
// unchanged:
//
//	export const frontmatter   // { title, description, ... }
//	export const toc           // [ { title, url:"#id", depth }, ... ]
//	export const structuredData// { headings:[...], contents:[...] } (search)
//	export default MDXContent  // the React body component (page.data.body)
//
// Prose is rendered to chroma-highlighted HTML (injected via
// dangerouslySetInnerHTML). A bounded set of fumadocs-ui block components is also
// supported so docs can use real blocks:
//   - Callouts:  GitHub admonition blockquotes ("> [!NOTE]" / TIP / WARNING / …)
//   - Cards:     a contiguous <Cards>…<Card title="" href="">…</Card>…</Cards> block
// Arbitrary JSX/ESM inside content is intentionally out of scope — this is a
// Markdown compiler with a small block allowlist, not a general MDX transpiler.
package mdx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// Options configures a compile. It is intentionally small for now.
type Options struct {
	// Development enables non-minified, source-friendly output (reserved).
	Development bool
}

// TOCItem is a single table-of-contents entry, matching fumadocs' page.data.toc
// element shape.
type TOCItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`   // fragment link, e.g. "#installation"
	Depth int    `json:"depth"` // heading level (2 = h2, 3 = h3, ...)
}

// Heading is one indexed heading in the search payload.
type Heading struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// Content is one indexed text block in the search payload.
type Content struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

// StructuredData mirrors fumadocs-core's Orama search payload.
type StructuredData struct {
	Headings []Heading `json:"headings"`
	Contents []Content `json:"contents"`
}

// Result is the output of compiling one content file.
type Result struct {
	TSX            string
	Frontmatter    map[string]any
	TOC            []TOCItem
	StructuredData StructuredData
	HTML           string // full rendered body HTML (for tests/tools)
}

var (
	admonitionRe = regexp.MustCompile(`(?s)^\s*\[!(\w+)\]\s*`)
	// cardRe matches a paired <Card ...>description</Card> (attributes in group 1).
	cardRe = regexp.MustCompile(`(?is)<Card\b([^>]*)>(.*?)</Card>`)
	attrRe = regexp.MustCompile(`(\w+)\s*=\s*"([^"]*)"`)
	tagRe  = regexp.MustCompile(`(?s)<[^>]+>`)
)

func newMarkdown() goldmark.Markdown {
	// Fenced code blocks are handled separately (see buildSegments) so they can be
	// highlighted into fumadocs/shiki-compatible markup and wrapped in fumadocs-ui's
	// CodeBlock. goldmark here handles everything else (GFM + auto heading IDs).
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)
}

// Compile transpiles a Markdown/MDX source file into a React module.
func Compile(src []byte, opts Options) (Result, error) {
	body, fm, err := splitFrontmatter(src)
	if err != nil {
		return Result{}, err
	}

	md := newMarkdown()
	doc := md.Parser().Parse(text.NewReader(body))

	toc, sd := extractTOCAndData(doc, body)

	// Full HTML render (exposed for tests/tools).
	var full bytes.Buffer
	if err := md.Renderer().Render(&full, body, doc); err != nil {
		return Result{}, fmt.Errorf("mdx: render markdown: %w", err)
	}

	segments, err := buildSegments(doc, body, md)
	if err != nil {
		return Result{}, err
	}

	res := Result{Frontmatter: fm, TOC: toc, StructuredData: sd, HTML: full.String()}
	tsx, err := generateModule(res, segments)
	if err != nil {
		return Result{}, err
	}
	res.TSX = tsx
	return res, nil
}

// ── segments ────────────────────────────────────────────────────────────────

type segKind int

const (
	segHTML segKind = iota
	segCallout
	segCards
	segCode
)

type cardData struct {
	Title       string
	Href        string
	Description string
}

type segment struct {
	kind        segKind
	html        string     // segHTML / segCallout inner html
	calloutType string     // segCallout
	cards       []cardData // segCards
	codeInner   string     // segCode: <code> inner HTML (shiki spans)
}

// buildSegments walks the document's top-level blocks, grouping ordinary prose
// into HTML segments and lifting admonition blockquotes and <Cards> blocks into
// real fumadocs-ui component segments.
func buildSegments(doc ast.Node, source []byte, md goldmark.Markdown) ([]segment, error) {
	var segs []segment
	var htmlBuf bytes.Buffer

	flush := func() {
		if htmlBuf.Len() > 0 {
			segs = append(segs, segment{kind: segHTML, html: htmlBuf.String()})
			htmlBuf.Reset()
		}
	}

	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		// Admonition blockquote → Callout.
		if bq, ok := n.(*ast.Blockquote); ok {
			if typ, inner, isCallout := renderCallout(bq, source, md); isCallout {
				flush()
				segs = append(segs, segment{kind: segCallout, calloutType: typ, html: inner})
				continue
			}
		}
		// <Cards> HTML block → Cards/Card.
		if hb, ok := n.(*ast.HTMLBlock); ok {
			raw := htmlBlockText(hb, source)
			if strings.Contains(raw, "<Cards") {
				if cards := parseCards(raw); len(cards) > 0 {
					flush()
					segs = append(segs, segment{kind: segCards, cards: cards})
					continue
				}
			}
		}
		// Fenced code block → fumadocs CodeBlock with shiki-compatible markup.
		if fc, ok := n.(*ast.FencedCodeBlock); ok {
			flush()
			inner := highlightShiki(fencedCodeText(fc, source), string(fc.Language(source)))
			segs = append(segs, segment{kind: segCode, codeInner: inner})
			continue
		}
		// Ordinary block → render to HTML and accumulate.
		if err := md.Renderer().Render(&htmlBuf, source, n); err != nil {
			return nil, fmt.Errorf("mdx: render block: %w", err)
		}
	}
	flush()
	return segs, nil
}

// renderCallout detects a GitHub-style admonition blockquote and returns the
// fumadocs Callout type plus the blockquote's inner HTML (marker removed).
func renderCallout(bq *ast.Blockquote, source []byte, md goldmark.Markdown) (typ, inner string, ok bool) {
	para, isPara := bq.FirstChild().(*ast.Paragraph)
	if !isPara {
		return "", "", false
	}
	m := admonitionRe.FindStringSubmatch(nodeText(para, source))
	if m == nil {
		return "", "", false
	}
	var b bytes.Buffer
	if err := md.Renderer().Render(&b, source, bq); err != nil {
		return "", "", false
	}
	h := strings.TrimSpace(b.String())
	h = strings.TrimPrefix(h, "<blockquote>")
	h = strings.TrimSuffix(h, "</blockquote>")
	h = admonitionRe.ReplaceAllString(strings.TrimSpace(h), "")
	// The marker sits inside the first <p>; strip it there too.
	h = strings.Replace(h, "[!"+m[1]+"]", "", 1)
	return calloutType(m[1]), strings.TrimSpace(h), true
}

// calloutType maps an admonition marker to a fumadocs Callout type.
func calloutType(marker string) string {
	switch strings.ToUpper(marker) {
	case "TIP", "HINT", "SUCCESS":
		return "success"
	case "WARNING", "WARN", "CAUTION":
		return "warn"
	case "DANGER", "ERROR":
		return "error"
	case "IDEA":
		return "idea"
	default: // NOTE, INFO, IMPORTANT, …
		return "info"
	}
}

// parseCards extracts <Card title="" href="">description</Card> entries from a
// raw <Cards> HTML block.
func parseCards(raw string) []cardData {
	var out []cardData
	for _, m := range cardRe.FindAllStringSubmatch(raw, -1) {
		attrs := map[string]string{}
		for _, a := range attrRe.FindAllStringSubmatch(m[1], -1) {
			attrs[strings.ToLower(a[1])] = a[2]
		}
		desc := strings.TrimSpace(tagRe.ReplaceAllString(m[2], ""))
		if attrs["description"] != "" {
			desc = attrs["description"]
		}
		if attrs["title"] == "" {
			continue
		}
		out = append(out, cardData{Title: attrs["title"], Href: attrs["href"], Description: desc})
	}
	return out
}

func htmlBlockText(n *ast.HTMLBlock, source []byte) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	return b.String()
}

func fencedCodeText(n *ast.FencedCodeBlock, source []byte) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// ── module generation ─────────────────────────────────────────────────────────

// generateModule renders the final ES module source for one page.
func generateModule(res Result, segments []segment) (string, error) {
	fmJSON, err := jsonConst(res.Frontmatter)
	if err != nil {
		return "", err
	}
	tocJSON, err := jsonConst(res.TOC)
	if err != nil {
		return "", err
	}
	sdJSON, err := jsonConst(res.StructuredData)
	if err != nil {
		return "", err
	}

	usesCallout, usesCards, usesCode := false, false, false
	for _, s := range segments {
		switch s.kind {
		case segCallout:
			usesCallout = true
		case segCards:
			usesCards = true
		case segCode:
			usesCode = true
		}
	}

	var b strings.Builder
	b.WriteString(`import { createElement, Fragment } from "react";` + "\n")
	// Import block components through Pola's "use client" re-export so their
	// browser-only deps (e.g. lucide-react) never enter the server JS engine.
	if usesCallout {
		b.WriteString(`import { Callout } from "@pola/fumadocs/blocks";` + "\n")
	}
	if usesCards {
		b.WriteString(`import { Card, Cards } from "@pola/fumadocs/blocks";` + "\n")
	}
	if usesCode {
		b.WriteString(`import { CodeBlock, Pre } from "@pola/fumadocs/blocks";` + "\n")
	}
	b.WriteString("export const frontmatter = " + fmJSON + ";\n")
	b.WriteString("export const toc = " + tocJSON + ";\n")
	b.WriteString("export const structuredData = " + sdJSON + ";\n")

	exprs := make([]string, 0, len(segments))
	for i, s := range segments {
		expr, err := segmentExpr(i, s)
		if err != nil {
			return "", err
		}
		exprs = append(exprs, expr)
	}

	b.WriteString("export default function MDXContent() {\n")
	if len(exprs) == 0 {
		b.WriteString("  return null;\n")
	} else {
		b.WriteString("  return createElement(Fragment, null,\n    " + strings.Join(exprs, ",\n    ") + "\n  );\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

// segmentExpr renders one segment to a createElement expression.
func segmentExpr(i int, s segment) (string, error) {
	switch s.kind {
	case segHTML:
		htmlJSON, err := jsonConst(escapeNonASCII(s.html))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(`createElement("div", { key: %d, className: "pola-mdx", dangerouslySetInnerHTML: { __html: %s } })`, i, htmlJSON), nil
	case segCallout:
		htmlJSON, err := jsonConst(escapeNonASCII(s.html))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(
			`createElement(Callout, { key: %d, type: %q }, createElement("div", { className: "pola-mdx", dangerouslySetInnerHTML: { __html: %s } }))`,
			i, s.calloutType, htmlJSON), nil
	case segCards:
		cardExprs := make([]string, 0, len(s.cards))
		for j, c := range s.cards {
			title, _ := jsonConst(escapeNonASCII(c.Title))
			desc, _ := jsonConst(escapeNonASCII(c.Description))
			href, _ := jsonConst(c.Href)
			cardExprs = append(cardExprs, fmt.Sprintf(
				`createElement(Card, { key: %d, title: %s, href: %s }, %s)`, j, title, href, desc))
		}
		return fmt.Sprintf(`createElement(Cards, { key: %d }, %s)`, i, strings.Join(cardExprs, ", ")), nil
	case segCode:
		innerJSON, err := jsonConst(escapeNonASCII(s.codeInner))
		if err != nil {
			return "", err
		}
		// fumadocs-ui CodeBlock (figure + border + copy button + bg-fd-card
		// background, matching fumadocs' default) wrapping fumadocs Pre and a shiki
		// <code>; token colours come from fumadocs' own shiki.css via the
		// --shiki-light/--shiki-dark vars on each span.
		return fmt.Sprintf(
			`createElement(CodeBlock, { key: %d }, createElement(Pre, null, createElement("code", { className: "shiki", dangerouslySetInnerHTML: { __html: %s } })))`,
			i, innerJSON), nil
	}
	return "null", nil
}

// ── frontmatter ───────────────────────────────────────────────────────────────

func splitFrontmatter(src []byte) ([]byte, map[string]any, error) {
	s := src
	if !bytes.HasPrefix(s, []byte("---\n")) && !bytes.HasPrefix(s, []byte("---\r\n")) {
		return src, nil, nil
	}
	rest := s[len("---"):]
	idx := indexClosingFence(rest)
	if idx < 0 {
		return src, nil, nil
	}
	fmBytes := rest[:idx]
	body := stripClosingFence(rest[idx:])

	fm := map[string]any{}
	if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
		return nil, nil, fmt.Errorf("mdx: parse frontmatter: %w", err)
	}
	return body, fm, nil
}

func indexClosingFence(rest []byte) int {
	off := 0
	for off < len(rest) {
		nl := bytes.IndexByte(rest[off:], '\n')
		var line []byte
		if nl < 0 {
			line = rest[off:]
		} else {
			line = rest[off : off+nl]
		}
		if strings.TrimRight(string(line), "\r") == "---" {
			return off
		}
		if nl < 0 {
			break
		}
		off += nl + 1
	}
	return -1
}

func stripClosingFence(body []byte) []byte {
	nl := bytes.IndexByte(body, '\n')
	if nl < 0 {
		return nil
	}
	return body[nl+1:]
}

// ── TOC + search extraction ───────────────────────────────────────────────────

func extractTOCAndData(doc ast.Node, src []byte) ([]TOCItem, StructuredData) {
	var toc []TOCItem
	sd := StructuredData{Headings: []Heading{}, Contents: []Content{}}
	currentHeading := ""

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Heading:
			title := nodeText(node, src)
			id := headingID(node)
			if node.Level >= 2 && node.Level <= 4 {
				toc = append(toc, TOCItem{Title: title, URL: "#" + id, Depth: node.Level})
			}
			sd.Headings = append(sd.Headings, Heading{ID: id, Content: title})
			currentHeading = id
			return ast.WalkSkipChildren, nil
		case *ast.Paragraph:
			// Strip a leading admonition marker ("[!NOTE]") so it doesn't pollute
			// the search index.
			content := strings.TrimSpace(admonitionRe.ReplaceAllString(nodeText(node, src), ""))
			if content != "" {
				sd.Contents = append(sd.Contents, Content{Heading: currentHeading, Content: content})
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return toc, sd
}

func headingID(h *ast.Heading) string {
	if v, ok := h.AttributeString("id"); ok {
		switch t := v.(type) {
		case []byte:
			return string(t)
		case string:
			return t
		}
	}
	return ""
}

// nodeText collects the concatenated plain-text content of a node subtree. A
// single ast.Walk already descends into every child (emphasis, links, inline
// code, etc.), so text is gathered by appending the leaf Text/String segments —
// no manual recursion (which would re-walk the same subtree and overflow).
func nodeText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// ── helpers ───────────────────────────────────────────────────────────────────

// escapeNonASCII replaces every non-ASCII rune with its numeric HTML entity
// (&#N;), leaving ASCII (including HTML markup and existing entities) untouched.
// Intended for HTML consumed via dangerouslySetInnerHTML, so the browser decodes
// the entities while the intermediate string that flows through Pola's RSC Flight
// stream stays pure ASCII (a raw multi-byte char otherwise triggers an "Invalid
// code point" error in the streaming layer).
func escapeNonASCII(s string) string {
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r > 0x7f {
			fmt.Fprintf(&b, "&#%d;", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// jsonConst marshals v to a JS-embeddable JSON literal.
func jsonConst(v any) (string, error) {
	if v == nil {
		return "null", nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("mdx: marshal generated value: %w", err)
	}
	return string(data), nil
}
