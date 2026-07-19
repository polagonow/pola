package mdx

import (
	"fmt"
	"html"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Light/dark chroma styles used to derive per-token colours. fumadocs-ui's
// shiki.css reads --shiki-light / --shiki-dark and picks the right one per theme,
// so emitting both keeps code blocks styled by fumadocs' own CSS (no custom
// stylesheet needed) in both light and dark mode.
var (
	shikiLight = styles.Get("github")
	shikiDark  = styles.Get("github-dark")
)

// highlightShiki tokenises code with chroma and renders shiki-compatible markup:
// `.line`-wrapped spans carrying --shiki-light / --shiki-dark CSS variables. It
// returns the inner HTML for a <code class="shiki">. The container background is
// left to fumadocs-ui's CodeBlock (bg-fd-card), matching fumadocs' default look;
// only the token colours come from these palettes.
func highlightShiki(code, lang string) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return `<span class="line">` + html.EscapeString(code) + `</span>`
	}

	var b strings.Builder
	lineSlices := chroma.SplitTokensIntoLines(it.Tokens())
	for i, line := range lineSlices {
		b.WriteString(`<span class="line">`)
		for _, tok := range line {
			text := html.EscapeString(strings.TrimSuffix(tok.Value, "\n"))
			lc := colourOf(shikiLight, tok.Type)
			dc := colourOf(shikiDark, tok.Type)
			if lc == "" && dc == "" {
				b.WriteString(text)
				continue
			}
			b.WriteString(`<span style="`)
			if lc != "" {
				fmt.Fprintf(&b, "--shiki-light:%s;", lc)
			}
			if dc != "" {
				fmt.Fprintf(&b, "--shiki-dark:%s;", dc)
			}
			b.WriteString(`">`)
			b.WriteString(text)
			b.WriteString(`</span>`)
		}
		b.WriteString(`</span>`)
		if i < len(lineSlices)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func colourOf(s *chroma.Style, t chroma.TokenType) string {
	if s == nil {
		return ""
	}
	if e := s.Get(t); e.Colour.IsSet() {
		return e.Colour.String()
	}
	return ""
}
