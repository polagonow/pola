package mdx

import (
	"strings"
	"testing"
)

func TestCompileFrontmatterAndExports(t *testing.T) {
	src := []byte(`---
title: Getting Started
description: How to begin
---

# Getting Started

Intro paragraph.

## Installation

Run the installer.

## Usage

Use it well.
`)

	res, err := Compile(src, Options{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if got := res.Frontmatter["title"]; got != "Getting Started" {
		t.Errorf("frontmatter title = %v, want Getting Started", got)
	}
	if got := res.Frontmatter["description"]; got != "How to begin" {
		t.Errorf("frontmatter description = %v, want How to begin", got)
	}

	// TOC should contain the two h2 headings with fragment URLs.
	if len(res.TOC) != 2 {
		t.Fatalf("toc len = %d, want 2: %+v", len(res.TOC), res.TOC)
	}
	if res.TOC[0].Title != "Installation" || res.TOC[0].URL != "#installation" || res.TOC[0].Depth != 2 {
		t.Errorf("toc[0] = %+v, want {Installation #installation 2}", res.TOC[0])
	}
	if res.TOC[1].Title != "Usage" || res.TOC[1].URL != "#usage" {
		t.Errorf("toc[1] = %+v, want Usage/#usage", res.TOC[1])
	}

	// Structured data should include headings and content blocks for search.
	if len(res.StructuredData.Headings) < 3 {
		t.Errorf("structuredData headings = %d, want >=3", len(res.StructuredData.Headings))
	}
	if len(res.StructuredData.Contents) < 3 {
		t.Errorf("structuredData contents = %d, want >=3", len(res.StructuredData.Contents))
	}

	// Generated module must carry the four fumadocs-contract exports + body.
	for _, want := range []string{
		"export const frontmatter =",
		"export const toc =",
		"export const structuredData =",
		"export default function MDXContent",
		`import { createElement, Fragment } from "react"`,
	} {
		if !strings.Contains(res.TSX, want) {
			t.Errorf("generated module missing %q\n---\n%s", want, res.TSX)
		}
	}
}

func TestCompileNoFrontmatter(t *testing.T) {
	res, err := Compile([]byte("# Title\n\nHello **world**.\n"), Options{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if res.Frontmatter != nil {
		t.Errorf("frontmatter = %v, want nil", res.Frontmatter)
	}
	if !strings.Contains(res.HTML, "<strong>world</strong>") {
		t.Errorf("html missing bold render: %s", res.HTML)
	}
}

func TestCompileInlineCodeNoOverflow(t *testing.T) {
	// Inline code spans previously triggered infinite recursion in nodeText.
	src := []byte("## Using `foo`\n\nCall `bar()` then `baz` and `qux`.\n")
	res, err := Compile(src, Options{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(res.TOC) != 1 || res.TOC[0].Title != "Using foo" {
		t.Errorf("toc = %+v, want single 'Using foo'", res.TOC)
	}
	if !strings.Contains(res.HTML, "<code>bar()</code>") {
		t.Errorf("inline code not rendered: %s", res.HTML)
	}
}

func TestCompileCodeHighlighting(t *testing.T) {
	src := []byte("```go\npackage main\n\nfunc main() {}\n```\n")
	res, err := Compile(src, Options{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Code fences render through fumadocs-ui's CodeBlock with shiki-compatible
	// per-token --shiki-light / --shiki-dark variables (styled by fumadocs' CSS).
	if !strings.Contains(res.TSX, `import { CodeBlock, Pre } from "@pola/fumadocs/blocks"`) {
		t.Errorf("CodeBlock import missing: %s", res.TSX)
	}
	if !strings.Contains(res.TSX, `createElement(CodeBlock,`) {
		t.Errorf("CodeBlock element missing: %s", res.TSX)
	}
	if !strings.Contains(res.TSX, `--shiki-light:`) {
		t.Errorf("shiki per-token colours missing: %s", res.TSX)
	}
}

func TestCompileEscapesNonASCII(t *testing.T) {
	// An em dash in the body must be emitted as an HTML entity so the RSC Flight
	// stream stays pure ASCII.
	res, err := Compile([]byte("Alpha — beta café.\n"), Options{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// The em dash must be encoded as an HTML entity in the body (JSON-escaped, so
	// the "&" appears as & in the module source; JS/the browser decode it).
	if !strings.Contains(res.TSX, `#8212;`) {
		t.Errorf("em dash not escaped to an HTML entity in body html: %s", res.TSX)
	}
	// Every dangerouslySetInnerHTML body — the fields that reach the Flight stream
	// — must contain no raw non-ASCII bytes (that would break RSC streaming in the
	// VM). structuredData may keep raw text; it is never passed to a component.
	found := false
	for _, l := range strings.Split(res.TSX, "\n") {
		if !strings.Contains(l, "dangerouslySetInnerHTML") {
			continue
		}
		found = true
		for i := 0; i < len(l); i++ {
			if l[i] >= 0x80 {
				t.Fatalf("body html contains raw non-ASCII byte at %d: %s", i, l)
			}
		}
	}
	if !found {
		t.Fatal("no dangerouslySetInnerHTML body in generated module")
	}
}

func TestCompileCallout(t *testing.T) {
	src := []byte("# T\n\n> [!WARNING]\n> Be **careful** here.\n\nAfter.\n")
	res, err := Compile(src, Options{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !strings.Contains(res.TSX, `import { Callout } from "@pola/fumadocs/blocks"`) {
		t.Errorf("Callout import missing: %s", res.TSX)
	}
	if !strings.Contains(res.TSX, `createElement(Callout, { key: `) || !strings.Contains(res.TSX, `type: "warn"`) {
		t.Errorf("Callout element/type missing: %s", res.TSX)
	}
	// The [!WARNING] marker must not leak into the generated module.
	if strings.Contains(res.TSX, "[!WARNING]") {
		t.Errorf("admonition marker leaked into output: %s", res.TSX)
	}
	// The inner markdown is rendered (checked on the un-escaped full HTML).
	if !strings.Contains(res.HTML, "<strong>careful</strong>") {
		t.Errorf("callout inner markdown not rendered: %s", res.HTML)
	}
}

func TestCompileCards(t *testing.T) {
	src := []byte("# T\n\n<Cards>\n<Card title=\"Alpha\" href=\"/a\">First card</Card>\n<Card title=\"Beta\" href=\"/b\">Second card</Card>\n</Cards>\n")
	res, err := Compile(src, Options{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !strings.Contains(res.TSX, `import { Card, Cards } from "@pola/fumadocs/blocks"`) {
		t.Errorf("Card import missing: %s", res.TSX)
	}
	if !strings.Contains(res.TSX, `createElement(Cards, `) {
		t.Errorf("Cards element missing: %s", res.TSX)
	}
	if !strings.Contains(res.TSX, `title: "Alpha"`) || !strings.Contains(res.TSX, `href: "/a"`) {
		t.Errorf("Card props missing: %s", res.TSX)
	}
	if !strings.Contains(res.TSX, `"First card"`) {
		t.Errorf("Card description missing: %s", res.TSX)
	}
}

func TestCompileGFMTable(t *testing.T) {
	src := []byte("| A | B |\n|---|---|\n| 1 | 2 |\n")
	res, err := Compile(src, Options{})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !strings.Contains(res.HTML, "<table>") {
		t.Errorf("GFM table not rendered: %s", res.HTML)
	}
}
