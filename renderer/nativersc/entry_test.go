package nativersc

import (
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
)

// TestGenerateEntry verifies the generated server entry exposes the Go-driven
// primitives and, crucially, does NOT pull in react-server-dom-webpack's
// renderToReadableStream (the whole point of nativersc).
func TestGenerateEntry(t *testing.T) {
	gen := &EntryGenerator{}
	src, err := gen.Generate(EntryGenConfig{
		AppDir: "/tmp/app",
		Pages: []core.PageEntry{
			{PageComponentPath: "/tmp/app/app/page.tsx"},
		},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	mustContain := []string{
		`import React from "react";`,
		"const __pages__ = {",
		"Index,", // PageAlias for app/page.tsx with no segments
		"globalThis.__createRoot__ = function",
		"globalThis.__rsc__ =",
		`Symbol.for("react.client.reference")`,
		`Symbol.for("react.suspense")`,
	}
	for _, s := range mustContain {
		if !strings.Contains(src, s) {
			t.Errorf("generated entry missing %q\n---\n%s", s, src)
		}
	}

	mustNotContain := []string{
		"react-server-dom-webpack",
		"renderToReadableStream",
	}
	for _, s := range mustNotContain {
		if strings.Contains(src, s) {
			t.Errorf("generated entry should not contain %q\n---\n%s", s, src)
		}
	}
}

// TestGenerateEntryServerActions verifies the generated entry imports the
// discovered 'use server' modules and installs the registry + invocation
// helpers, keyed by moduleId:exportName.
func TestGenerateEntryServerActions(t *testing.T) {
	gen := &EntryGenerator{}
	src, err := gen.Generate(EntryGenConfig{
		AppDir: "/tmp/app",
		Pages:  []core.PageEntry{{PageComponentPath: "/tmp/app/app/page.tsx"}},
		ServerActions: []ServerActionModule{
			{ModuleID: "actions/todo", AbsPath: "/tmp/app/actions/todo.ts"},
		},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	mustContain := []string{
		`import * as __sa_0__ from "./actions/todo.ts";`,
		"globalThis.__POLA_SERVER_ACTIONS__",
		`__pola_register_actions__("actions/todo", __sa_0__);`,
		"globalThis.__getServerFunction__ = function",
		"globalThis.__invokeServerAction__ = function",
		`Symbol.for("react.server.reference")`,
	}
	for _, s := range mustContain {
		if !strings.Contains(src, s) {
			t.Errorf("generated entry missing %q\n---\n%s", s, src)
		}
	}
}

// TestGenerateEntryNoServerActions verifies the registry is omitted when no
// 'use server' modules are present.
func TestGenerateEntryNoServerActions(t *testing.T) {
	gen := &EntryGenerator{}
	src, err := gen.Generate(EntryGenConfig{
		AppDir: "/tmp/app",
		Pages:  []core.PageEntry{{PageComponentPath: "/tmp/app/app/page.tsx"}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if strings.Contains(src, "__POLA_SERVER_ACTIONS__") {
		t.Errorf("registry should be omitted when there are no server actions\n%s", src)
	}
}

// TestGenerateEntryShellExtractor verifies the shell extractor is emitted only
// when the root layout returns <html>.
func TestGenerateEntryShellExtractor(t *testing.T) {
	gen := &EntryGenerator{}

	withHTML, err := gen.Generate(EntryGenConfig{
		AppDir:                "/tmp/app",
		Pages:                 []core.PageEntry{{PageComponentPath: "/tmp/app/app/page.tsx"}},
		RootLayoutReturnsHTML: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withHTML, "__extractShell__") {
		t.Error("expected __extractShell__ when RootLayoutReturnsHTML=true")
	}

	without, err := gen.Generate(EntryGenConfig{
		AppDir: "/tmp/app",
		Pages:  []core.PageEntry{{PageComponentPath: "/tmp/app/app/page.tsx"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(without, "__extractShell__") {
		t.Error("did not expect __extractShell__ when RootLayoutReturnsHTML=false")
	}
}
