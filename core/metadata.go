package core

import "strings"

// Title holds the document title. Resolution order:
//  1. Absolute — rendered verbatim; Template is ignored.
//  2. Default + Template — Template's %s is replaced by Default.
//  3. Default — rendered as-is.
//  4. Template — rendered as-is (root-layout fallback).
//  5. All empty — <title> tag is omitted.
type Title struct {
	Absolute string
	Default  string
	Template string // use %s as the page-title placeholder
}

// ResolveTitle returns the string to use inside <title>, applying NextJS-style
// template substitution (%s → page title). Returns "" when all fields are empty.
func ResolveTitle(t Title) string {
	if t.Absolute != "" {
		return t.Absolute
	}
	if t.Default != "" && t.Template != "" {
		return strings.Replace(t.Template, "%s", t.Default, 1)
	}
	if t.Default != "" {
		return t.Default
	}
	return t.Template
}

// Author represents a document author.
type Author struct {
	Name *string
	URL  *string
}

// Icon describes a single <link> icon element.
type Icon struct {
	URL   string  // href value; required
	Sizes *string // e.g. "32x32", "any"
	Type  *string // e.g. "image/png", "image/svg+xml"
}

// OtherIcon is an icon with an explicit rel attribute.
type OtherIcon struct {
	Rel  string
	Icon Icon
}

// Icons groups icon variants by their <link rel="..."> value.
type Icons struct {
	Icon     []Icon      // rel="icon"
	Shortcut []Icon      // rel="shortcut icon"
	Apple    []Icon      // rel="apple-touch-icon"
	Other    []OtherIcon // custom rel values
}

// OpenGraphImage represents a set of og:image meta tags.
type OpenGraphImage struct {
	URL       string  `meta:"property=og:image"`
	SecureURL *string `meta:"property=og:image:secure_url"`
	Type      *string `meta:"property=og:image:type"`
	Alt       *string `meta:"property=og:image:alt"`
	Width     *int    `meta:"property=og:image:width"`
	Height    *int    `meta:"property=og:image:height"`
}

// OpenGraph holds Open Graph protocol metadata.
// When Type is nil the renderer defaults to "website".
type OpenGraph struct {
	Type        *string          `meta:"property=og:type" default:"website"`
	Title       *string          `meta:"property=og:title"`
	Description *string          `meta:"property=og:description"`
	URL         *string          `meta:"property=og:url"`
	SiteName    *string          `meta:"property=og:site_name"`
	Locale      *string          `meta:"property=og:locale"`
	Images      []OpenGraphImage // rendered via slice-of-struct recursion
}

// Twitter card type constants.
const (
	TwitterCardSummary           = "summary"
	TwitterCardSummaryLargeImage = "summary_large_image"
	TwitterCardApp               = "app"
	TwitterCardPlayer            = "player"
)

// TwitterMeta holds Twitter Card metadata.
type TwitterMeta struct {
	Card        *string  `meta:"name=twitter:card"`
	Site        *string  `meta:"name=twitter:site"`
	Creator     *string  `meta:"name=twitter:creator"`
	Title       *string  `meta:"name=twitter:title"`
	Description *string  `meta:"name=twitter:description"`
	Images      []string `meta:"name=twitter:image"`
}

// Robots controls search-engine crawling and indexing directives.
// Nil fields are omitted from the rendered content attribute.
type Robots struct {
	Index           *bool
	Follow          *bool
	NoArchive       *bool
	NoSnippet       *bool
	NoImageIndex    *bool
	NoCache         *bool
	NoTranslate     *bool
	MaxSnippet      *int    // -1 = unlimited
	MaxImagePreview *string // "none", "standard", "large"
	MaxVideoPreview *int    // -1 = unlimited
}

// AlternateURLs declares canonical and hreflang alternate links.
type AlternateURLs struct {
	Canonical *string
	Languages map[string]string // BCP-47 language tag → URL
}

// Verification holds site-verification meta tag values.
type Verification struct {
	Google *string `meta:"name=google-site-verification"`
	Yandex *string `meta:"name=yandex-verification"`
	Yahoo  *string `meta:"name=y_key"`
	Me     *string `link:"rel=me"`
}

// Metadata is the top-level SEO metadata struct, mirroring the NextJS
// Metadata type. Nil pointer fields are silently omitted from the rendered HTML.
type Metadata struct {
	Title           Title    // custom: ResolveTitle()
	Description     *string  `meta:"name=description"`
	ApplicationName *string  `meta:"name=application-name"`
	Authors         []Author // custom: mixed meta+link per element
	Generator       *string  `meta:"name=generator"`
	Keywords        []string `meta:"name=keywords" sep:", "`
	Referrer        *string  `meta:"name=referrer"`
	Creator         *string  `meta:"name=creator"`
	Publisher       *string  `meta:"name=publisher"`

	// SEO
	Robots       *Robots        // custom: HeadRenderer
	Alternates   *AlternateURLs // custom: HeadRenderer
	Verification *Verification  // tagged sub-struct

	// Icons
	Icons *Icons // custom: HeadRenderer

	// Social
	OpenGraph *OpenGraph   // tagged sub-struct + Images recursion
	Twitter   *TwitterMeta // tagged sub-struct

	// Other renders as <meta name="KEY" content="VALUE"> for each entry.
	Other map[string]string `meta:"name=*"`
}
