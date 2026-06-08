package serveraction

import "testing"

func TestHasUseServerDirective(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want bool
	}{
		{"double quotes", `"use server"
export async function a() {}`, true},
		{"single quotes", `'use server'
export async function a() {}`, true},
		{"with semicolon", `"use server";
export async function a() {}`, true},
		{"leading whitespace", "\n\n   'use server'\n", true},
		{"after line comment", "// top\n'use server'\n", true},
		{"after block comment", "/* a\n b */ 'use server'\n", true},
		{"after use strict", `"use strict";
"use server";
export function a() {}`, true},
		{"bom prefix", string([]byte{0xEF, 0xBB, 0xBF}) + "'use server'\n", true},
		{"not present", `export async function a() {}`, false},
		{"use client", `'use client'
export default function C() {}`, false},
		{"directive after code", `const x = 1;
'use server'`, false},
		{"inside function", `export function a() {
  'use server'
}`, false},
		{"substring not directive", `const s = "this will use server stuff"`, false},
		{"empty", ``, false},
		{"unterminated literal", `'use server`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasUseServerDirective([]byte(c.src)); got != c.want {
				t.Errorf("HasUseServerDirective(%q) = %v, want %v", c.src, got, c.want)
			}
		})
	}
}
