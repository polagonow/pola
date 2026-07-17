package goja_test

import (
	"strings"
	"testing"

	"github.com/polagonow/pola/engine/goja"
)

// A regex Goja's RE2 engine rejects (invalid character-class range) — the exact
// class of failure that stock npm deps like lucide-react can trigger.
const re2BadBundle = `var re = /[\s-_]/; var __render__ = function(){ return re; };`

func TestCompile_RE2Diagnostic(t *testing.T) {
	_, err := goja.New(re2BadBundle)
	if err == nil {
		t.Fatal("expected a compile error for an RE2-incompatible regex")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--vm quickjs") {
		t.Fatalf("diagnostic should suggest the --vm escape hatch; got:\n%s", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "regular expression") {
		t.Fatalf("diagnostic should mention the regex cause; got:\n%s", msg)
	}
}

func TestCompile_NonRegexError_Unchanged(t *testing.T) {
	_, err := goja.New(`function ( { syntax error`)
	if err == nil {
		t.Fatal("expected a syntax error")
	}
	if strings.Contains(err.Error(), "--vm quickjs") {
		t.Fatalf("non-regex errors must not get the RE2 hint; got:\n%s", err.Error())
	}
}
