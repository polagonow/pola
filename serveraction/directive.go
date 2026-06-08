// Package serveraction implements RSC-style server actions for Pola: discovery
// of 'use server' modules, request validation/sanitization, and the HTTP
// handlers that invoke registered actions in the renderer's JS runtime pool.
//
// The runtime semantics mirror rari (registry keyed "moduleId:exportName",
// rari-style lookup, arg validation, CSRF origin checks, and a
// {success,result,error,redirect} response envelope), but discovery happens at
// build time because Pola compiles a single JS bundle rather than importing
// modules dynamically at request time.
package serveraction

// HasUseServerDirective reports whether src begins — after an optional UTF-8
// BOM, leading whitespace, and // line or /* */ block comments — with a
// module-level "use server" / 'use server' directive in the directive prologue.
//
// Only the prologue counts: a 'use server' string appearing after real code (or
// inside a function body) does not make the module a server-action module. Other
// prologue directives (e.g. "use strict") are skipped so that
// `"use strict"; "use server";` is still recognized.
func HasUseServerDirective(src []byte) bool {
	// Strip a leading UTF-8 BOM (EF BB BF) if present.
	if len(src) >= 3 && src[0] == 0xEF && src[1] == 0xBB && src[2] == 0xBF {
		src = src[3:]
	}
	s := string(src)
	i := 0
	n := len(s)
	for i < n {
		i = skipTrivia(s, i)
		if i >= n {
			break
		}
		q := s[i]
		if q != '"' && q != '\'' {
			// First meaningful token is not a string literal → no prologue.
			return false
		}
		// Read the string literal, honoring backslash escapes.
		j := i + 1
		for j < n && s[j] != q {
			if s[j] == '\\' {
				j++
			}
			j++
		}
		if j >= n {
			return false // unterminated literal
		}
		if s[i+1:j] == "use server" {
			return true
		}
		i = j + 1
		// Consume an optional trailing semicolon before the next directive.
		i = skipTrivia(s, i)
		if i < n && s[i] == ';' {
			i++
		}
	}
	return false
}

// skipTrivia advances past ASCII whitespace and // line / /* */ block comments,
// returning the index of the next significant byte.
func skipTrivia(s string, i int) int {
	n := len(s)
	for i < n {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '/' && i+1 < n && s[i+1] == '/':
			i += 2
			for i < n && s[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && s[i+1] == '*':
			i += 2
			for i+1 < n && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i += 2
		default:
			return i
		}
	}
	return i
}
