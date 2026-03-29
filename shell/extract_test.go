package shell

import (
	"testing"
)

func TestExtractDocumentProps_FullDocument(t *testing.T) {
	html := `<html lang="fr" dir="rtl"><head><link rel="preconnect" href="https://fonts.googleapis.com"/><meta name="theme-color" content="#000"/></head><body class="dark antialiased"><header>My Site</header><main><!--POLA_CHILDREN--></main><footer>© 2026</footer></body></html>`

	dp, err := ExtractDocumentProps(html)
	if err != nil {
		t.Fatal(err)
	}

	if dp.HTMLAttributes["lang"] != "fr" {
		t.Errorf("html lang = %q, want %q", dp.HTMLAttributes["lang"], "fr")
	}
	if dp.HTMLAttributes["dir"] != "rtl" {
		t.Errorf("html dir = %q, want %q", dp.HTMLAttributes["dir"], "rtl")
	}
	if dp.BodyAttributes["class"] != "dark antialiased" {
		t.Errorf("body class = %q, want %q", dp.BodyAttributes["class"], "dark antialiased")
	}
	// <link> stays in HeadElements, <meta name="theme-color"> goes to MetaOverrides
	if len(dp.HeadElements) != 1 {
		t.Fatalf("head elements count = %d, want 1; got %v", len(dp.HeadElements), dp.HeadElements)
	}
	if dp.MetaOverrides["theme-color"] != "#000" {
		t.Errorf("meta theme-color = %q, want %q", dp.MetaOverrides["theme-color"], "#000")
	}
	if dp.BodyPrefix == "" {
		t.Error("body prefix is empty, expected header content")
	}
	if dp.BodySuffix == "" {
		t.Error("body suffix is empty, expected footer content")
	}
}

func TestExtractDocumentProps_NoPlaceholder(t *testing.T) {
	html := `<html lang="en"><head></head><body class="test"><p>hello</p></body></html>`

	dp, err := ExtractDocumentProps(html)
	if err != nil {
		t.Fatal(err)
	}

	if dp.HTMLAttributes["lang"] != "en" {
		t.Errorf("html lang = %q, want %q", dp.HTMLAttributes["lang"], "en")
	}
	if dp.BodyAttributes["class"] != "test" {
		t.Errorf("body class = %q, want %q", dp.BodyAttributes["class"], "test")
	}
	if dp.BodyPrefix == "" {
		t.Error("body prefix is empty, expected all body content")
	}
	if dp.BodySuffix != "" {
		t.Errorf("body suffix = %q, want empty", dp.BodySuffix)
	}
}

func TestExtractDocumentProps_EmptyHead(t *testing.T) {
	html := `<html><head></head><body><!--POLA_CHILDREN--></body></html>`

	dp, err := ExtractDocumentProps(html)
	if err != nil {
		t.Fatal(err)
	}

	if len(dp.HeadElements) != 0 {
		t.Errorf("head elements = %v, want empty", dp.HeadElements)
	}
	if dp.Title != "" {
		t.Errorf("title = %q, want empty", dp.Title)
	}
	if dp.Charset != "" {
		t.Errorf("charset = %q, want empty", dp.Charset)
	}
	if dp.BodyPrefix != "" {
		t.Errorf("body prefix = %q, want empty", dp.BodyPrefix)
	}
	if dp.BodySuffix != "" {
		t.Errorf("body suffix = %q, want empty", dp.BodySuffix)
	}
}

func TestExtractDocumentProps_NoHTML(t *testing.T) {
	html := `<div>not a document</div>`

	dp, err := ExtractDocumentProps(html)
	if err != nil {
		t.Fatal(err)
	}

	// html.Parse always creates an implicit <html> node
	if dp == nil {
		t.Fatal("expected non-nil DocumentProps")
	}
}

func TestExtractDocumentProps_ClassifyHead(t *testing.T) {
	html := `<html><head>
		<title>My App</title>
		<meta charset="utf-8"/>
		<meta name="viewport" content="width=device-width"/>
		<meta name="description" content="A blog about things"/>
		<meta name="theme-color" content="#fff"/>
		<meta property="og:title" content="OG Title"/>
		<link rel="icon" href="/favicon.ico"/>
		<link rel="stylesheet" href="/styles.css"/>
		<script src="/analytics.js"></script>
	</head><body><!--POLA_CHILDREN--></body></html>`

	dp, err := ExtractDocumentProps(html)
	if err != nil {
		t.Fatal(err)
	}

	// Title extracted
	if dp.Title != "My App" {
		t.Errorf("title = %q, want %q", dp.Title, "My App")
	}

	// Charset extracted
	if dp.Charset != "utf-8" {
		t.Errorf("charset = %q, want %q", dp.Charset, "utf-8")
	}

	// Viewport extracted
	if dp.Viewport != "width=device-width" {
		t.Errorf("viewport = %q, want %q", dp.Viewport, "width=device-width")
	}

	// Named metas go to MetaOverrides
	if dp.MetaOverrides["description"] != "A blog about things" {
		t.Errorf("meta description = %q, want %q", dp.MetaOverrides["description"], "A blog about things")
	}
	if dp.MetaOverrides["theme-color"] != "#fff" {
		t.Errorf("meta theme-color = %q, want %q", dp.MetaOverrides["theme-color"], "#fff")
	}

	// OG property meta stays in HeadElements as raw HTML
	// <link> and <script> stay in HeadElements
	// Expected: og:title meta, favicon link, stylesheet link, script = 4
	if len(dp.HeadElements) != 4 {
		t.Errorf("head elements count = %d, want 4; got %v", len(dp.HeadElements), dp.HeadElements)
	}
}

func TestExtractDocumentProps_OnlyFrameworkTags(t *testing.T) {
	html := `<html><head>
		<title>Just Title</title>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width,initial-scale=1"/>
	</head><body><!--POLA_CHILDREN--></body></html>`

	dp, err := ExtractDocumentProps(html)
	if err != nil {
		t.Fatal(err)
	}

	if dp.Title != "Just Title" {
		t.Errorf("title = %q, want %q", dp.Title, "Just Title")
	}
	if dp.Charset != "UTF-8" {
		t.Errorf("charset = %q, want %q", dp.Charset, "UTF-8")
	}
	if dp.Viewport != "width=device-width,initial-scale=1" {
		t.Errorf("viewport = %q, want %q", dp.Viewport, "width=device-width,initial-scale=1")
	}
	// No remaining head elements
	if len(dp.HeadElements) != 0 {
		t.Errorf("head elements = %v, want empty", dp.HeadElements)
	}
	if len(dp.MetaOverrides) != 0 {
		t.Errorf("meta overrides = %v, want empty", dp.MetaOverrides)
	}
}
