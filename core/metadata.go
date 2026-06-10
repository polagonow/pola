package core

import (
	"encoding/json"
	"strings"
)

// Title holds the document title. Resolution order:
//  1. Absolute — rendered verbatim; Template is ignored.
//  2. Default + Template — Template's %s is replaced by Default.
//  3. Default — rendered as-is.
//  4. Template — rendered as-is (root-layout fallback).
//  5. All empty — <title> tag is omitted.
type Title struct {
	Absolute string `json:"absolute,omitempty"`
	Default  string `json:"default,omitempty"`
	Template string `json:"template,omitempty"` // use %s as the page-title placeholder
}

// UnmarshalJSON handles both string ("My Page") and object ({default, template, absolute}) forms.
func (t *Title) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.Default = s
		return nil
	}
	type alias Title
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*t = Title(raw)
	return nil
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
	Name *string `json:"name,omitempty"`
	URL  *string `json:"url,omitempty"`
}

// Icon describes a single <link> icon element.
type Icon struct {
	URL   string  `json:"url"`            // href value; required
	Sizes *string `json:"sizes,omitempty"` // e.g. "32x32", "any"
	Type  *string `json:"type,omitempty"`  // e.g. "image/png", "image/svg+xml"
}

// OtherIcon is an icon with an explicit rel attribute.
type OtherIcon struct {
	Rel  string `json:"rel"`
	Icon Icon   `json:"icon"`
}

// Icons groups icon variants by their <link rel="..."> value.
type Icons struct {
	Icon     []Icon      `json:"icon,omitempty"`     // rel="icon"
	Shortcut []Icon      `json:"shortcut,omitempty"` // rel="shortcut icon"
	Apple    []Icon      `json:"apple,omitempty"`    // rel="apple-touch-icon"
	Other    []OtherIcon `json:"other,omitempty"`    // custom rel values
}

// OpenGraphImage represents a set of og:image meta tags.
type OpenGraphImage struct {
	URL       string  `meta:"property=og:image" json:"url"`
	SecureURL *string `meta:"property=og:image:secure_url" json:"secureUrl,omitempty"`
	Type      *string `meta:"property=og:image:type" json:"type,omitempty"`
	Alt       *string `meta:"property=og:image:alt" json:"alt,omitempty"`
	Width     *int    `meta:"property=og:image:width" json:"width,omitempty"`
	Height    *int    `meta:"property=og:image:height" json:"height,omitempty"`
}

// OpenGraph holds Open Graph protocol metadata.
// When Type is nil the renderer defaults to "website".
type OpenGraph struct {
	Type        *string          `meta:"property=og:type" default:"website" json:"type,omitempty"`
	Title       *string          `meta:"property=og:title" json:"title,omitempty"`
	Description *string          `meta:"property=og:description" json:"description,omitempty"`
	URL         *string          `meta:"property=og:url" json:"url,omitempty"`
	SiteName    *string          `meta:"property=og:site_name" json:"siteName,omitempty"`
	Locale      *string          `meta:"property=og:locale" json:"locale,omitempty"`
	Images      []OpenGraphImage `json:"images,omitempty"` // rendered via slice-of-struct recursion
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
	Card        *string  `meta:"name=twitter:card" json:"card,omitempty"`
	Site        *string  `meta:"name=twitter:site" json:"site,omitempty"`
	Creator     *string  `meta:"name=twitter:creator" json:"creator,omitempty"`
	Title       *string  `meta:"name=twitter:title" json:"title,omitempty"`
	Description *string  `meta:"name=twitter:description" json:"description,omitempty"`
	Images      []string `meta:"name=twitter:image" json:"images,omitempty"`
}

// Robots controls search-engine crawling and indexing directives.
// Nil fields are omitted from the rendered content attribute.
type Robots struct {
	Index           *bool   `json:"index,omitempty"`
	Follow          *bool   `json:"follow,omitempty"`
	NoArchive       *bool   `json:"noarchive,omitempty"`
	NoSnippet       *bool   `json:"nosnippet,omitempty"`
	NoImageIndex    *bool   `json:"noimageindex,omitempty"`
	NoCache         *bool   `json:"nocache,omitempty"`
	NoTranslate     *bool   `json:"notranslate,omitempty"`
	MaxSnippet      *int    `json:"maxSnippet,omitempty"`      // -1 = unlimited
	MaxImagePreview *string `json:"maxImagePreview,omitempty"` // "none", "standard", "large"
	MaxVideoPreview *int    `json:"maxVideoPreview,omitempty"` // -1 = unlimited
}

// AlternateURLs declares canonical and hreflang alternate links.
type AlternateURLs struct {
	Canonical *string           `json:"canonical,omitempty"`
	Languages map[string]string `json:"languages,omitempty"` // BCP-47 language tag → URL
}

// Verification holds site-verification meta tag values.
type Verification struct {
	Google *string `meta:"name=google-site-verification" json:"google,omitempty"`
	Yandex *string `meta:"name=yandex-verification" json:"yandex,omitempty"`
	Yahoo  *string `meta:"name=y_key" json:"yahoo,omitempty"`
	Me     *string `link:"rel=me" json:"me,omitempty"`
}

// Metadata is the top-level SEO metadata struct, mirroring the NextJS
// Metadata type. Nil pointer fields are silently omitted from the rendered HTML.
type Metadata struct {
	Title           Title    `json:"title,omitempty"`           // custom: ResolveTitle()
	Description     *string  `meta:"name=description" json:"description,omitempty"`
	ApplicationName *string  `meta:"name=application-name" json:"applicationName,omitempty"`
	Authors         []Author `json:"authors,omitempty"` // custom: mixed meta+link per element
	Generator       *string  `meta:"name=generator" json:"generator,omitempty"`
	Keywords        []string `meta:"name=keywords" sep:", " json:"keywords,omitempty"`
	Referrer        *string  `meta:"name=referrer" json:"referrer,omitempty"`
	Creator         *string  `meta:"name=creator" json:"creator,omitempty"`
	Publisher       *string  `meta:"name=publisher" json:"publisher,omitempty"`

	// SEO
	Robots       *Robots        `json:"robots,omitempty"`       // custom: HeadRenderer
	Alternates   *AlternateURLs `json:"alternates,omitempty"`   // custom: HeadRenderer
	Verification *Verification  `json:"verification,omitempty"` // tagged sub-struct

	// Icons
	Icons *Icons `json:"icons,omitempty"` // custom: HeadRenderer

	// Social
	OpenGraph *OpenGraph   `json:"openGraph,omitempty"` // tagged sub-struct + Images recursion
	Twitter   *TwitterMeta `json:"twitter,omitempty"`   // tagged sub-struct

	// Other renders as <meta name="KEY" content="VALUE"> for each entry.
	Other map[string]string `meta:"name=*" json:"other,omitempty"`
}
