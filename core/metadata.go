package core

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
	URL       string
	SecureURL *string // og:image:secure_url
	Type      *string // og:image:type  e.g. "image/jpeg"
	Width     *int
	Height    *int
	Alt       *string // og:image:alt
}

// OpenGraph holds Open Graph protocol metadata.
// When Type is nil the renderer defaults to "website".
type OpenGraph struct {
	Title       *string
	Description *string
	URL         *string
	SiteName    *string
	Images      []OpenGraphImage
	Locale      *string // e.g. "en_US"
	Type        *string // "website", "article", "book", "profile", …
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
	Card        *string  // one of the TwitterCard* constants
	Site        *string  // @username of the website
	Creator     *string  // @username of the content creator
	Title       *string
	Description *string
	Images      []string // twitter:image URLs
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
	Google *string
	Yandex *string
	Yahoo  *string
	Me     *string // rendered as <link rel="me" href="...">
}

// Metadata is the top-level SEO metadata struct, mirroring the NextJS
// Metadata type. Nil pointer fields are silently omitted from the rendered HTML.
type Metadata struct {
	Title           Title
	Description     *string
	ApplicationName *string
	Authors         []Author
	Generator       *string
	Keywords        []string
	Referrer        *string // referrer policy value
	Creator         *string
	Publisher       *string

	// SEO
	Robots       *Robots
	Alternates   *AlternateURLs
	Verification *Verification

	// Icons
	Icons *Icons

	// Social
	OpenGraph *OpenGraph
	Twitter   *TwitterMeta

	// Other renders as <meta name="KEY" content="VALUE"> for each entry.
	Other map[string]string
}
