package core

import (
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestRenderMetaHTML_Nil(t *testing.T) {
	if got := RenderMetaHTML(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestRenderMetaHTML_Empty(t *testing.T) {
	if got := RenderMetaHTML(&Metadata{}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestRenderMetaHTML_Title(t *testing.T) {
	tests := []struct {
		name  string
		title Title
		want  string
	}{
		{"absolute", Title{Absolute: "Abs"}, "<title>Abs</title>"},
		{"default+template", Title{Default: "Page", Template: "%s | Site"}, "<title>Page | Site</title>"},
		{"default only", Title{Default: "Page"}, "<title>Page</title>"},
		{"template only", Title{Template: "Site"}, "<title>Site</title>"},
		{"empty", Title{}, ""},
		{"escaping", Title{Default: `<script>alert("xss")</script>`}, `<title>&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;</title>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderMetaHTML(&Metadata{Title: tt.title})
			if got != tt.want {
				t.Errorf("\ngot  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestRenderMetaHTML_SimpleFields(t *testing.T) {
	m := &Metadata{
		Description:     ptr("A blog about things"),
		ApplicationName: ptr("MyApp"),
		Generator:       ptr("Pola v1"),
		Referrer:        ptr("no-referrer"),
		Creator:         ptr("Alice"),
		Publisher:       ptr("Bob"),
	}
	got := RenderMetaHTML(m)

	expects := []string{
		`<meta name="description" content="A blog about things"/>`,
		`<meta name="application-name" content="MyApp"/>`,
		`<meta name="generator" content="Pola v1"/>`,
		`<meta name="referrer" content="no-referrer"/>`,
		`<meta name="creator" content="Alice"/>`,
		`<meta name="publisher" content="Bob"/>`,
	}
	for _, e := range expects {
		if !strings.Contains(got, e) {
			t.Errorf("missing %s in:\n%s", e, got)
		}
	}
}

func TestRenderMetaHTML_Keywords(t *testing.T) {
	m := &Metadata{Keywords: []string{"go", "react", "ssr"}}
	got := RenderMetaHTML(m)
	want := `<meta name="keywords" content="go, react, ssr"/>`
	if !strings.Contains(got, want) {
		t.Errorf("missing %s in:\n%s", want, got)
	}
}

func TestRenderMetaHTML_Authors(t *testing.T) {
	m := &Metadata{
		Authors: []Author{
			{Name: ptr("Alice"), URL: ptr("https://alice.dev")},
			{Name: ptr("Bob")},
		},
	}
	got := RenderMetaHTML(m)

	if !strings.Contains(got, `<link rel="author" href="https://alice.dev"/>`) {
		t.Errorf("missing author link in:\n%s", got)
	}
	if !strings.Contains(got, `<meta name="author" content="Alice"/>`) {
		t.Errorf("missing author meta (Alice) in:\n%s", got)
	}
	if !strings.Contains(got, `<meta name="author" content="Bob"/>`) {
		t.Errorf("missing author meta (Bob) in:\n%s", got)
	}
}

func TestRenderMetaHTML_Robots(t *testing.T) {
	m := &Metadata{
		Robots: &Robots{
			Index:           ptr(true),
			Follow:          ptr(false),
			NoArchive:       ptr(true),
			MaxSnippet:      ptr(100),
			MaxImagePreview: ptr("large"),
		},
	}
	got := RenderMetaHTML(m)
	want := `<meta name="robots" content="index, nofollow, noarchive, max-snippet:100, max-image-preview:large"/>`
	if !strings.Contains(got, want) {
		t.Errorf("missing robots in:\n%s\nwant: %s", got, want)
	}
}

func TestRenderMetaHTML_Alternates(t *testing.T) {
	m := &Metadata{
		Alternates: &AlternateURLs{
			Canonical: ptr("https://example.com"),
			Languages: map[string]string{"en": "https://example.com/en", "fr": "https://example.com/fr"},
		},
	}
	got := RenderMetaHTML(m)

	if !strings.Contains(got, `<link rel="canonical" href="https://example.com"/>`) {
		t.Errorf("missing canonical in:\n%s", got)
	}
	if !strings.Contains(got, `hreflang="en"`) {
		t.Errorf("missing en hreflang in:\n%s", got)
	}
	if !strings.Contains(got, `hreflang="fr"`) {
		t.Errorf("missing fr hreflang in:\n%s", got)
	}
}

func TestRenderMetaHTML_Icons(t *testing.T) {
	m := &Metadata{
		Icons: &Icons{
			Icon: []Icon{
				{URL: "/favicon.ico"},
				{URL: "/icon-32.png", Sizes: ptr("32x32"), Type: ptr("image/png")},
			},
			Apple: []Icon{{URL: "/apple-touch-icon.png"}},
		},
	}
	got := RenderMetaHTML(m)

	if !strings.Contains(got, `<link rel="icon" href="/favicon.ico"/>`) {
		t.Errorf("missing icon link in:\n%s", got)
	}
	if !strings.Contains(got, `sizes="32x32"`) {
		t.Errorf("missing sizes attr in:\n%s", got)
	}
	if !strings.Contains(got, `type="image/png"`) {
		t.Errorf("missing type attr in:\n%s", got)
	}
	if !strings.Contains(got, `<link rel="apple-touch-icon" href="/apple-touch-icon.png"/>`) {
		t.Errorf("missing apple-touch-icon in:\n%s", got)
	}
}

func TestRenderMetaHTML_OpenGraph(t *testing.T) {
	m := &Metadata{
		OpenGraph: &OpenGraph{
			Title:       ptr("OG Title"),
			Description: ptr("OG Desc"),
			URL:         ptr("https://example.com"),
			SiteName:    ptr("Example"),
			Images: []OpenGraphImage{
				{URL: "https://example.com/img.jpg", Alt: ptr("Image"), Width: ptr(800), Height: ptr(600)},
			},
		},
	}
	got := RenderMetaHTML(m)

	expects := []string{
		`<meta property="og:type" content="website"/>`,
		`<meta property="og:title" content="OG Title"/>`,
		`<meta property="og:description" content="OG Desc"/>`,
		`<meta property="og:url" content="https://example.com"/>`,
		`<meta property="og:site_name" content="Example"/>`,
		`<meta property="og:image" content="https://example.com/img.jpg"/>`,
		`<meta property="og:image:alt" content="Image"/>`,
		`<meta property="og:image:width" content="800"/>`,
		`<meta property="og:image:height" content="600"/>`,
	}
	for _, e := range expects {
		if !strings.Contains(got, e) {
			t.Errorf("missing %s in:\n%s", e, got)
		}
	}
}

func TestRenderMetaHTML_OpenGraphTypeDefault(t *testing.T) {
	m := &Metadata{OpenGraph: &OpenGraph{Title: ptr("Test")}}
	got := RenderMetaHTML(m)
	if !strings.Contains(got, `<meta property="og:type" content="website"/>`) {
		t.Errorf("missing default og:type in:\n%s", got)
	}
}

func TestRenderMetaHTML_OpenGraphTypeCustom(t *testing.T) {
	m := &Metadata{OpenGraph: &OpenGraph{Type: ptr("article")}}
	got := RenderMetaHTML(m)
	if !strings.Contains(got, `<meta property="og:type" content="article"/>`) {
		t.Errorf("missing custom og:type in:\n%s", got)
	}
	if strings.Contains(got, `content="website"`) {
		t.Errorf("should not have default og:type when custom is set:\n%s", got)
	}
}

func TestRenderMetaHTML_Twitter(t *testing.T) {
	m := &Metadata{
		Twitter: &TwitterMeta{
			Card:        ptr("summary_large_image"),
			Site:        ptr("@example"),
			Title:       ptr("Tweet Title"),
			Description: ptr("Tweet Desc"),
			Images:      []string{"https://example.com/a.jpg", "https://example.com/b.jpg"},
		},
	}
	got := RenderMetaHTML(m)

	expects := []string{
		`<meta name="twitter:card" content="summary_large_image"/>`,
		`<meta name="twitter:site" content="@example"/>`,
		`<meta name="twitter:title" content="Tweet Title"/>`,
		`<meta name="twitter:description" content="Tweet Desc"/>`,
		`<meta name="twitter:image" content="https://example.com/a.jpg"/>`,
		`<meta name="twitter:image" content="https://example.com/b.jpg"/>`,
	}
	for _, e := range expects {
		if !strings.Contains(got, e) {
			t.Errorf("missing %s in:\n%s", e, got)
		}
	}
}

func TestRenderMetaHTML_Verification(t *testing.T) {
	m := &Metadata{
		Verification: &Verification{
			Google: ptr("google-123"),
			Yandex: ptr("yandex-456"),
			Yahoo:  ptr("yahoo-789"),
			Me:     ptr("https://me.example.com"),
		},
	}
	got := RenderMetaHTML(m)

	expects := []string{
		`<meta name="google-site-verification" content="google-123"/>`,
		`<meta name="yandex-verification" content="yandex-456"/>`,
		`<meta name="y_key" content="yahoo-789"/>`,
		`<link rel="me" href="https://me.example.com"/>`,
	}
	for _, e := range expects {
		if !strings.Contains(got, e) {
			t.Errorf("missing %s in:\n%s", e, got)
		}
	}
}

func TestRenderMetaHTML_Other(t *testing.T) {
	m := &Metadata{
		Other: map[string]string{"theme-color": "#000", "color-scheme": "dark"},
	}
	got := RenderMetaHTML(m)

	// Sorted order: color-scheme before theme-color
	if !strings.Contains(got, `<meta name="color-scheme" content="dark"/>`) {
		t.Errorf("missing color-scheme in:\n%s", got)
	}
	if !strings.Contains(got, `<meta name="theme-color" content="#000"/>`) {
		t.Errorf("missing theme-color in:\n%s", got)
	}
	csIdx := strings.Index(got, "color-scheme")
	tcIdx := strings.Index(got, "theme-color")
	if csIdx > tcIdx {
		t.Error("Other map should render in sorted key order")
	}
}

func TestRenderMetaHTML_FieldOrder(t *testing.T) {
	m := &Metadata{
		Title:       Title{Default: "Page"},
		Description: ptr("Desc"),
		Keywords:    []string{"a"},
		OpenGraph:   &OpenGraph{Title: ptr("OG")},
		Twitter:     &TwitterMeta{Card: ptr("summary")},
		Other:       map[string]string{"x": "y"},
	}
	got := RenderMetaHTML(m)

	// Verify ordering: title < description < keywords < og < twitter < other
	indices := []int{
		strings.Index(got, "<title>"),
		strings.Index(got, `name="description"`),
		strings.Index(got, `name="keywords"`),
		strings.Index(got, `property="og:`),
		strings.Index(got, `name="twitter:`),
		strings.Index(got, `name="x"`),
	}
	for i := 1; i < len(indices); i++ {
		if indices[i] < indices[i-1] {
			t.Errorf("field order violation at index %d: %v\nHTML: %s", i, indices, got)
			break
		}
	}
}

func TestRenderMetaHTML_HTMLEscaping(t *testing.T) {
	m := &Metadata{
		Description: ptr(`<script>alert("xss")</script>`),
	}
	got := RenderMetaHTML(m)
	if strings.Contains(got, "<script>") {
		t.Errorf("XSS in output:\n%s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("description not escaped:\n%s", got)
	}
}
