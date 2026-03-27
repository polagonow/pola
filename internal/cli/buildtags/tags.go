// Package buildtags computes Go build tags for the Pola framework.
package buildtags

import "strings"

// RuntimeTags returns build tags for dev/bundle runs (includes bundler + CSS).
func RuntimeTags(vm, bundler, renderer, router, css string) string {
	tags := []string{vm, bundler, renderer, router}
	if css != "" && css != "none" {
		tags = append(tags, css)
	}
	return strings.Join(tags, " ")
}

// BinaryTags returns build tags for production binaries.
// Excludes bundler and CSS — assets are pre-built and embedded.
func BinaryTags(vm, renderer, router string) string {
	return strings.Join([]string{"embed", vm, renderer, router}, " ")
}
