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
	if len(dp.HeadElements) != 2 {
		t.Fatalf("head elements count = %d, want 2", len(dp.HeadElements))
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
	// With no placeholder, everything goes to prefix
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
