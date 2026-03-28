package shell

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"

	"github.com/polagonow/pola/core"
)

// ChildrenPlaceholder is the comment marker injected as {children} when
// calling the root layout for shell extraction.
const ChildrenPlaceholder = "POLA_CHILDREN"

// ExtractDocumentProps parses an HTML string (produced by the JS shell
// extractor) and returns the document-level properties: <html> attributes,
// <body> attributes, <head> child elements, and body content before/after
// the <!--POLA_CHILDREN--> placeholder.
func ExtractDocumentProps(htmlStr string) (*core.DocumentProps, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil, err
	}

	dp := &core.DocumentProps{
		HTMLAttributes: make(map[string]string),
		BodyAttributes: make(map[string]string),
	}

	htmlNode := findElement(doc, "html")
	if htmlNode == nil {
		return dp, nil
	}
	dp.HTMLAttributes = collectAttrs(htmlNode)

	if headNode := findElement(htmlNode, "head"); headNode != nil {
		dp.HeadElements = renderChildren(headNode)
	}

	if bodyNode := findElement(htmlNode, "body"); bodyNode != nil {
		dp.BodyAttributes = collectAttrs(bodyNode)
		dp.BodyPrefix, dp.BodySuffix = splitOnPlaceholder(bodyNode)
	}

	return dp, nil
}

// findElement performs a shallow search for the first child element with the
// given tag name. It does not recurse deeply — it checks direct children and
// one level of implicit wrappers (html.Parse inserts <html>/<head>/<body>).
func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
		// html.Parse wraps content in implicit nodes; recurse one level.
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// collectAttrs extracts all attributes from an element node into a map.
func collectAttrs(n *html.Node) map[string]string {
	attrs := make(map[string]string, len(n.Attr))
	for _, a := range n.Attr {
		attrs[a.Key] = a.Val
	}
	return attrs
}

// renderChildren renders each child of n to an HTML string, skipping
// whitespace-only text nodes. Returns one string per child element.
func renderChildren(n *html.Node) []string {
	var elems []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" {
			continue
		}
		s := renderNode(c)
		if s = strings.TrimSpace(s); s != "" {
			elems = append(elems, s)
		}
	}
	return elems
}

// splitOnPlaceholder renders all body children to HTML, then splits the
// string on the <!--POLA_CHILDREN--> comment marker. The placeholder may
// be nested inside wrapper elements (e.g. <main>{children}</main>).
// If no placeholder is found, all content goes into prefix.
func splitOnPlaceholder(body *html.Node) (prefix, suffix string) {
	var buf bytes.Buffer
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&buf, c)
	}
	full := buf.String()
	marker := "<!--" + ChildrenPlaceholder + "-->"
	idx := strings.Index(full, marker)
	if idx < 0 {
		return strings.TrimSpace(full), ""
	}
	return strings.TrimSpace(full[:idx]), strings.TrimSpace(full[idx+len(marker):])
}

// renderNode renders a single node to its HTML representation.
func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	_ = html.Render(&buf, n)
	return buf.String()
}
