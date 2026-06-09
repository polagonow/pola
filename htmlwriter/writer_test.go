package htmlwriter

import (
	"bytes"
	"math"
	"testing"
	"time"
)

func TestWriteText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"escapes <", "a < b", "a &lt; b"},
		{"escapes >", "a > b", "a &gt; b"},
		{"escapes &", "a & b", "a &amp; b"},
		{"escapes double quote", `say "hi"`, "say &#34;hi&#34;"},
		{"escapes single quote", "it's", "it&#39;s"},
		{"escapes all together", `<script>alert("xss")</script>`, "&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := New(&buf)
			_, err := w.WriteText(tt.input)
			if err != nil {
				t.Fatalf("WriteText(%q) error: %v", tt.input, err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("WriteText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteRaw(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	raw := `<div class="test">hello</div>`
	_, err := w.WriteRaw(raw)
	if err != nil {
		t.Fatalf("WriteRaw error: %v", err)
	}
	if got := buf.String(); got != raw {
		t.Errorf("WriteRaw = %q, want %q", got, raw)
	}
}

func TestWriteAttrNil(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.WriteAttr("hidden", nil); err != nil {
		t.Fatalf("WriteAttr(nil) error: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Errorf("WriteAttr(nil) = %q, want empty", got)
	}
}

func TestWriteAttrBool(t *testing.T) {
	tests := []struct {
		name  string
		attr  string
		value bool
		want  string
	}{
		{"regular true", "disabled", true, " disabled"},
		{"regular false", "disabled", false, ""},
		{"enumerable draggable true", "draggable", true, ` draggable="true"`},
		{"enumerable draggable false", "draggable", false, ` draggable="false"`},
		{"enumerable spellcheck true", "spellcheck", true, ` spellcheck="true"`},
		{"enumerable spellcheck false", "spellcheck", false, ` spellcheck="false"`},
		{"enumerable contenteditable true", "contenteditable", true, ` contenteditable="true"`},
		{"aria prefix true", "aria-hidden", true, ` aria-hidden="true"`},
		{"aria prefix false", "aria-hidden", false, ` aria-hidden="false"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := New(&buf)
			if err := w.WriteAttr(tt.attr, tt.value); err != nil {
				t.Fatalf("WriteAttr(%q, %v) error: %v", tt.attr, tt.value, err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("WriteAttr(%q, %v) = %q, want %q", tt.attr, tt.value, got, tt.want)
			}
		})
	}
}

func TestWriteAttrString(t *testing.T) {
	tests := []struct {
		name  string
		attr  string
		value string
		want  string
	}{
		{"plain", "id", "main", ` id="main"`},
		{"with quotes", "title", `say "hi"`, ` title="say &#34;hi&#34;"`},
		{"with angle brackets", "data-x", "<b>bold</b>", ` data-x="&lt;b&gt;bold&lt;/b&gt;"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := New(&buf)
			if err := w.WriteAttr(tt.attr, tt.value); err != nil {
				t.Fatalf("WriteAttr(%q, %q) error: %v", tt.attr, tt.value, err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("WriteAttr(%q, %q) = %q, want %q", tt.attr, tt.value, got, tt.want)
			}
		})
	}
}

func TestWriteAttrStyleMap(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	style := map[string]any{
		"backgroundColor": "red",
		"fontSize":        "14px",
	}
	if err := w.WriteAttr("style", style); err != nil {
		t.Fatalf("WriteAttr style error: %v", err)
	}
	got := buf.String()
	want := ` style="background-color:red;font-size:14px"`
	if got != want {
		t.Errorf("WriteAttr style = %q, want %q", got, want)
	}
}

func TestWriteAttrStyleEscaping(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	style := map[string]any{
		"content": `"hello;world"`,
	}
	if err := w.WriteAttr("style", style); err != nil {
		t.Fatalf("WriteAttr style escaping error: %v", err)
	}
	got := buf.String()
	want := ` style="content:&quot;hello\;world&quot;"`
	if got != want {
		t.Errorf("WriteAttr style escaping = %q, want %q", got, want)
	}
}

func TestWriteAttrClassSlice(t *testing.T) {
	tests := []struct {
		name  string
		value []any
		want  string
	}{
		{"simple classes", []any{"foo", "bar"}, ` class="foo bar"`},
		{"filters nil", []any{"foo", nil, "bar"}, ` class="foo bar"`},
		{"filters false", []any{"foo", false, "bar"}, ` class="foo bar"`},
		{"filters zero int", []any{"active", 0, "visible"}, ` class="active visible"`},
		{"all falsy", []any{nil, false, 0}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := New(&buf)
			if err := w.WriteAttr("class", tt.value); err != nil {
				t.Fatalf("WriteAttr class error: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("WriteAttr class = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteAttrNonClassSlice(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.WriteAttr("data-ids", []any{"a", "b", "c"}); err != nil {
		t.Fatalf("WriteAttr slice error: %v", err)
	}
	got := buf.String()
	want := ` data-ids="a,b,c"`
	if got != want {
		t.Errorf("WriteAttr slice = %q, want %q", got, want)
	}
}

func TestWriteAttrNumeric(t *testing.T) {
	tests := []struct {
		name  string
		attr  string
		value any
		want  string
	}{
		{"int", "tabindex", 3, ` tabindex="3"`},
		{"int64", "data-big", int64(9999999999), ` data-big="9999999999"`},
		{"float64", "data-ratio", 3.14, ` data-ratio="3.14"`},
		{"float64 infinity", "data-x", math.Inf(1), ` data-x="Infinity"`},
		{"float64 neg infinity", "data-x", math.Inf(-1), ` data-x="-Infinity"`},
		{"float64 NaN", "data-x", math.NaN(), ` data-x="NaN"`},
		{"uint", "data-count", uint(42), ` data-count="42"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := New(&buf)
			if err := w.WriteAttr(tt.attr, tt.value); err != nil {
				t.Fatalf("WriteAttr(%q, %v) error: %v", tt.attr, tt.value, err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("WriteAttr(%q, %v) = %q, want %q", tt.attr, tt.value, got, tt.want)
			}
		})
	}
}

func TestWriteAttrTime(t *testing.T) {
	ts := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.WriteAttr("datetime", ts); err != nil {
		t.Fatalf("WriteAttr time error: %v", err)
	}
	got := buf.String()
	want := ` datetime="2024-06-15T12:30:00Z"`
	if got != want {
		t.Errorf("WriteAttr time = %q, want %q", got, want)
	}
}

type testStringer struct{ val string }

func (s testStringer) String() string { return s.val }

func TestWriteAttrStringer(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.WriteAttr("data-val", testStringer{val: `<b>"test"</b>`}); err != nil {
		t.Fatalf("WriteAttr stringer error: %v", err)
	}
	got := buf.String()
	want := ` data-val="&lt;b&gt;&#34;test&#34;&lt;/b&gt;"`
	if got != want {
		t.Errorf("WriteAttr stringer = %q, want %q", got, want)
	}
}

func TestWriteAttrSpreadMap(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	spread := map[string]any{
		"id":    "main",
		"class": "container",
	}
	if err := w.WriteAttr("spread", spread); err != nil {
		t.Fatalf("WriteAttr spread error: %v", err)
	}
	got := buf.String()
	want := ` class="container" id="main"`
	if got != want {
		t.Errorf("WriteAttr spread = %q, want %q", got, want)
	}
}

func TestWriteAttrsSkipsChildren(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	attrs := map[string]any{
		"id":       "main",
		"children": "should be skipped",
		"class":    "active",
	}
	if err := w.WriteAttrs(attrs); err != nil {
		t.Fatalf("WriteAttrs error: %v", err)
	}
	got := buf.String()
	want := ` class="active" id="main"`
	if got != want {
		t.Errorf("WriteAttrs = %q, want %q", got, want)
	}
}

func TestWriteAttrsSorted(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	attrs := map[string]any{
		"z-index": "1",
		"alpha":   "first",
		"middle":  "center",
	}
	if err := w.WriteAttrs(attrs); err != nil {
		t.Fatalf("WriteAttrs error: %v", err)
	}
	got := buf.String()
	want := ` alpha="first" middle="center" z-index="1"`
	if got != want {
		t.Errorf("WriteAttrs sorted = %q, want %q", got, want)
	}
}

func TestOpenTag(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	attrs := map[string]any{
		"class": "container",
		"id":    "main",
	}
	if err := w.OpenTag("div", attrs); err != nil {
		t.Fatalf("OpenTag error: %v", err)
	}
	got := buf.String()
	want := `<div class="container" id="main">`
	if got != want {
		t.Errorf("OpenTag = %q, want %q", got, want)
	}
}

func TestOpenTagNoAttrs(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.OpenTag("div", nil); err != nil {
		t.Fatalf("OpenTag error: %v", err)
	}
	if got := buf.String(); got != "<div>" {
		t.Errorf("OpenTag no attrs = %q, want %q", got, "<div>")
	}
}

func TestCloseTag(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.CloseTag("div"); err != nil {
		t.Fatalf("CloseTag error: %v", err)
	}
	if got := buf.String(); got != "</div>" {
		t.Errorf("CloseTag = %q, want %q", got, "</div>")
	}
}

func TestVoidTag(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	attrs := map[string]any{
		"src": "image.png",
		"alt": "A photo",
	}
	if err := w.VoidTag("img", attrs); err != nil {
		t.Fatalf("VoidTag error: %v", err)
	}
	got := buf.String()
	want := `<img alt="A photo" src="image.png" />`
	if got != want {
		t.Errorf("VoidTag = %q, want %q", got, want)
	}
}

func TestVoidTagNoAttrs(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.VoidTag("br", nil); err != nil {
		t.Fatalf("VoidTag error: %v", err)
	}
	if got := buf.String(); got != "<br />" {
		t.Errorf("VoidTag no attrs = %q, want %q", got, "<br />")
	}
}

func TestXSSPreventionText(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	_, err := w.WriteText(`<script>alert("xss")</script>`)
	if err != nil {
		t.Fatalf("WriteText XSS error: %v", err)
	}
	got := buf.String()
	if got == `<script>alert("xss")</script>` {
		t.Error("WriteText did not escape script tags")
	}
	want := `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`
	if got != want {
		t.Errorf("WriteText XSS = %q, want %q", got, want)
	}
}

func TestXSSPreventionEventHandler(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.WriteAttr("onclick", `alert("xss")`); err != nil {
		t.Fatalf("WriteAttr XSS error: %v", err)
	}
	got := buf.String()
	want := ` onclick="alert(&#34;xss&#34;)"`
	if got != want {
		t.Errorf("WriteAttr XSS = %q, want %q", got, want)
	}
}

func TestIsVoidElement(t *testing.T) {
	voids := []string{"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr"}
	for _, tag := range voids {
		if !IsVoidElement(tag) {
			t.Errorf("IsVoidElement(%q) = false, want true", tag)
		}
	}
	nonVoids := []string{"div", "span", "p", "a", "script", "style"}
	for _, tag := range nonVoids {
		if IsVoidElement(tag) {
			t.Errorf("IsVoidElement(%q) = true, want false", tag)
		}
	}
}

func TestCamelToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"backgroundColor", "background-color"},
		{"fontSize", "font-size"},
		{"margin", "margin"},
		{"borderTopLeftRadius", "border-top-left-radius"},
		{"WebkitTransform", "-webkit-transform"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := camelToKebabCase(tt.input); got != tt.want {
				t.Errorf("camelToKebabCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteAttrEmptyStyleMap(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	if err := w.WriteAttr("style", map[string]any{}); err != nil {
		t.Fatalf("WriteAttr empty style error: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Errorf("WriteAttr empty style = %q, want empty", got)
	}
}
