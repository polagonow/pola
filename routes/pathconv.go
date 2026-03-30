package routes

import "strings"

// pathToPattern converts a Go package path to a URL pattern using the
// livebud alternating-segment convention.
//
// It finds "routes" in the import path, takes everything after it, and
// applies the resource/param alternation: nested resources under a registered
// resource get a ":id" param segment inserted.
//
// Examples (given registered packages for "users" and "users/posts"):
//
//	"myapp/routes"              → "/"
//	"myapp/routes/users"        → "/users"
//	"myapp/routes/users/posts"  → "/users/:id/posts"
//	"myapp/routes/auth/login"   → "/auth/login"
//
// The registeredPkgs set contains the relative paths (after "routes/") of all
// registered route packages, used to determine which segments are resources
// (have a route.go) vs. namespace-only directories.
//
// memberOverride, when non-nil, overrides the auto-detection for the immediate
// parent segment: true forces :id insertion, false suppresses it.
func pathToPattern(pkgPath string, registeredPkgs map[string]bool, memberOverride *bool) string {
	// Find "routes" segment and take everything after.
	parts := strings.Split(pkgPath, "/")
	routesIdx := -1
	for i, p := range parts {
		if p == "routes" {
			routesIdx = i
			break
		}
	}
	if routesIdx == -1 {
		return "/"
	}

	segments := parts[routesIdx+1:]
	if len(segments) == 0 {
		return "/"
	}

	return buildPattern(segments, registeredPkgs, memberOverride)
}

// buildPattern constructs a URL pattern from directory segments, inserting
// ":id" params after resource segments that have nested children.
//
// A segment is considered a "resource" if its relative path (up to and
// including itself) was registered as a route package. When a resource has
// a child segment, we insert an ":id" parameter between them.
//
// memberOverride applies to the immediate parent of the final segment only:
// true forces :id insertion, false suppresses it, nil uses auto-detection.
func buildPattern(segments []string, registeredPkgs map[string]bool, memberOverride *bool) string {
	var parts []string
	lastIdx := len(segments) - 1
	for i, seg := range segments {
		// Check if the previous segments (up to but not including current)
		// form a registered resource. If so, insert a param.
		if i > 0 {
			parentPath := strings.Join(segments[:i], "/")
			isMember := registeredPkgs[parentPath]

			// Apply memberOverride for the immediate parent of the final segment.
			if i == lastIdx && memberOverride != nil {
				isMember = *memberOverride
			}

			if isMember {
				paramName := singularize(segments[i-1]) + "_id"
				// First-level nesting uses just "id" for brevity.
				if i == 1 {
					paramName = "id"
				}
				parts = append(parts, ":"+paramName)
			}
		}
		parts = append(parts, seg)
	}
	return "/" + strings.Join(parts, "/")
}

// singularize is a naive English singularizer that handles common plural suffixes.
func singularize(s string) string {
	if strings.HasSuffix(s, "ies") && len(s) > 3 {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") || strings.HasSuffix(s, "xes") || strings.HasSuffix(s, "zes") || strings.HasSuffix(s, "ches") || strings.HasSuffix(s, "shes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}
