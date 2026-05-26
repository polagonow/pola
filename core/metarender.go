package core

import (
	"fmt"
	"html"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// HeadRenderer is implemented by types that need custom HTML head rendering
// beyond what struct tags can express.
type HeadRenderer interface {
	RenderHead(w *strings.Builder)
}

// RenderMetaHTML renders all <head> meta/link/title tags from a Metadata struct.
// It returns compact HTML with no whitespace between tags.
func RenderMetaHTML(m *Metadata) string {
	if m == nil {
		return ""
	}
	var w strings.Builder
	renderMetadataFields(&w, m)
	return w.String()
}

// renderMetadataFields walks the Metadata struct in field-declaration order,
// dispatching to tagged-field rendering, HeadRenderer, or custom handlers.
func renderMetadataFields(w *strings.Builder, m *Metadata) {
	t := reflect.TypeOf(*m)
	v := reflect.ValueOf(*m)

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)

		// Special cases by field name
		switch sf.Name {
		case "Title":
			if t := ResolveTitle(m.Title); t != "" {
				w.WriteString("<title>")
				w.WriteString(html.EscapeString(t))
				w.WriteString("</title>")
			}
			continue
		case "Authors":
			for _, a := range m.Authors {
				if a.URL != nil {
					writeLink(w, "author", *a.URL)
				}
				if a.Name != nil {
					writeMeta(w, "name", "author", *a.Name)
				}
			}
			continue
		}

		// Tagged fields
		if tag := sf.Tag.Get("meta"); tag != "" {
			renderMetaTag(w, sf, fv)
			continue
		}
		if tag := sf.Tag.Get("link"); tag != "" {
			renderLinkTag(w, sf, fv)
			continue
		}

		// Pointer to struct that implements HeadRenderer
		if fv.Kind() == reflect.Pointer && !fv.IsNil() {
			if hr, ok := fv.Interface().(HeadRenderer); ok {
				hr.RenderHead(w)
				continue
			}
		}

		// Pointer to tagged sub-struct (e.g. *OpenGraph, *TwitterMeta, *Verification)
		if fv.Kind() == reflect.Pointer && !fv.IsNil() {
			elem := fv.Elem()
			if elem.Kind() == reflect.Struct {
				renderTaggedStruct(w, elem)
				continue
			}
		}
	}
}

// renderTaggedStruct renders all tagged fields of a struct value.
func renderTaggedStruct(w *strings.Builder, v reflect.Value) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)

		if tag := sf.Tag.Get("meta"); tag != "" {
			renderMetaTag(w, sf, fv)
			continue
		}
		if tag := sf.Tag.Get("link"); tag != "" {
			renderLinkTag(w, sf, fv)
			continue
		}

		// Slice of tagged structs (e.g. []OpenGraphImage)
		if fv.Kind() == reflect.Slice && fv.Len() > 0 {
			elemType := fv.Type().Elem()
			if elemType.Kind() == reflect.Struct && hasMetaTags(elemType) {
				for j := 0; j < fv.Len(); j++ {
					renderTaggedStruct(w, fv.Index(j))
				}
				continue
			}
		}

		// Pointer to sub-struct implementing HeadRenderer
		if fv.Kind() == reflect.Pointer && !fv.IsNil() {
			if hr, ok := fv.Interface().(HeadRenderer); ok {
				hr.RenderHead(w)
				continue
			}
			elem := fv.Elem()
			if elem.Kind() == reflect.Struct {
				renderTaggedStruct(w, elem)
			}
		}
	}
}

// renderMetaTag emits a <meta> tag for a field with a `meta` struct tag.
func renderMetaTag(w *strings.Builder, sf reflect.StructField, fv reflect.Value) {
	tag := sf.Tag.Get("meta")
	attrKey, attrVal := parseTag(tag)
	defVal := sf.Tag.Get("default")

	switch fv.Kind() {
	case reflect.Pointer:
		if fv.IsNil() {
			if defVal != "" {
				writeMeta(w, attrKey, attrVal, defVal)
			}
			return
		}
		switch fv.Elem().Kind() {
		case reflect.String:
			writeMeta(w, attrKey, attrVal, fv.Elem().String())
		case reflect.Int:
			writeMeta(w, attrKey, attrVal, strconv.FormatInt(fv.Elem().Int(), 10))
		}

	case reflect.String:
		s := fv.String()
		if s == "" && defVal != "" {
			s = defVal
		}
		if s != "" {
			writeMeta(w, attrKey, attrVal, s)
		}

	case reflect.Slice:
		if fv.Type().Elem().Kind() == reflect.String {
			sep := sf.Tag.Get("sep")
			if sep != "" {
				// Join all elements into single tag
				parts := make([]string, fv.Len())
				for i := 0; i < fv.Len(); i++ {
					parts[i] = fv.Index(i).String()
				}
				if joined := strings.Join(parts, sep); joined != "" {
					writeMeta(w, attrKey, attrVal, joined)
				}
			} else {
				// One tag per element
				for i := 0; i < fv.Len(); i++ {
					writeMeta(w, attrKey, attrVal, fv.Index(i).String())
				}
			}
		}

	case reflect.Map:
		// map[string]string with meta:"name=*" → one tag per sorted key
		if attrVal == "*" && fv.Type().Key().Kind() == reflect.String {
			keys := make([]string, 0, fv.Len())
			for _, k := range fv.MapKeys() {
				keys = append(keys, k.String())
			}
			slices.Sort(keys)
			for _, k := range keys {
				writeMeta(w, attrKey, k, fv.MapIndex(reflect.ValueOf(k)).String())
			}
		}
	}
}

// renderLinkTag emits a <link> tag for a field with a `link` struct tag.
func renderLinkTag(w *strings.Builder, sf reflect.StructField, fv reflect.Value) {
	tag := sf.Tag.Get("link")
	_, attrVal := parseTag(tag)

	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return
		}
		if fv.Elem().Kind() == reflect.String {
			writeLink(w, attrVal, fv.Elem().String())
		}
	} else if fv.Kind() == reflect.String {
		if s := fv.String(); s != "" {
			writeLink(w, attrVal, s)
		}
	}
}

// parseTag splits "name=description" into ("name", "description").
func parseTag(tag string) (string, string) {
	if idx := strings.IndexByte(tag, '='); idx >= 0 {
		return tag[:idx], tag[idx+1:]
	}
	return tag, ""
}

// hasMetaTags checks if a struct type has any fields with meta tags.
func hasMetaTags(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Tag.Get("meta") != "" || t.Field(i).Tag.Get("link") != "" {
			return true
		}
	}
	return false
}

// writeMeta emits <meta attrKey="attrVal" content="content"/>.
func writeMeta(w *strings.Builder, attrKey, attrVal, content string) {
	w.WriteString("<meta ")
	w.WriteString(attrKey)
	w.WriteString(`="`)
	w.WriteString(html.EscapeString(attrVal))
	w.WriteString(`" content="`)
	w.WriteString(html.EscapeString(content))
	w.WriteString(`"/>`)
}

// writeLink emits <link rel="rel" href="href"/>.
func writeLink(w *strings.Builder, rel, href string) {
	w.WriteString(`<link rel="`)
	w.WriteString(html.EscapeString(rel))
	w.WriteString(`" href="`)
	w.WriteString(html.EscapeString(href))
	w.WriteString(`"/>`)
}

// --- HeadRenderer implementations ---

// RenderHead implements HeadRenderer for Robots.
func (r *Robots) RenderHead(w *strings.Builder) {
	if r == nil {
		return
	}
	rc := renderRobotsContent(r)
	if rc != "" {
		writeMeta(w, "name", "robots", rc)
	}
}

func renderRobotsContent(r *Robots) string {
	var parts []string

	addBool := func(trueVal, falseVal string, v *bool) {
		if v == nil {
			return
		}
		if *v {
			parts = append(parts, trueVal)
		} else {
			parts = append(parts, falseVal)
		}
	}
	addNoOnly := func(directive string, v *bool) {
		if v != nil && *v {
			parts = append(parts, directive)
		}
	}

	addBool("index", "noindex", r.Index)
	addBool("follow", "nofollow", r.Follow)
	addNoOnly("noarchive", r.NoArchive)
	addNoOnly("nosnippet", r.NoSnippet)
	addNoOnly("noimageindex", r.NoImageIndex)
	addNoOnly("nocache", r.NoCache)
	addNoOnly("notranslate", r.NoTranslate)

	if r.MaxSnippet != nil {
		parts = append(parts, fmt.Sprintf("max-snippet:%d", *r.MaxSnippet))
	}
	if r.MaxImagePreview != nil {
		parts = append(parts, fmt.Sprintf("max-image-preview:%s", *r.MaxImagePreview))
	}
	if r.MaxVideoPreview != nil {
		parts = append(parts, fmt.Sprintf("max-video-preview:%d", *r.MaxVideoPreview))
	}

	return strings.Join(parts, ", ")
}

// RenderHead implements HeadRenderer for AlternateURLs.
func (a *AlternateURLs) RenderHead(w *strings.Builder) {
	if a == nil {
		return
	}
	if a.Canonical != nil {
		writeLink(w, "canonical", *a.Canonical)
	}
	for _, lang := range slices.Sorted(maps.Keys(a.Languages)) {
		w.WriteString(`<link rel="alternate" hreflang="`)
		w.WriteString(html.EscapeString(lang))
		w.WriteString(`" href="`)
		w.WriteString(html.EscapeString(a.Languages[lang]))
		w.WriteString(`"/>`)
	}
}

// RenderHead implements HeadRenderer for Icons.
func (ic *Icons) RenderHead(w *strings.Builder) {
	if ic == nil {
		return
	}
	writeIconLinks(w, "icon", ic.Icon)
	writeIconLinks(w, "shortcut icon", ic.Shortcut)
	writeIconLinks(w, "apple-touch-icon", ic.Apple)
	for _, oi := range ic.Other {
		writeIconLinks(w, oi.Rel, []Icon{oi.Icon})
	}
}

func writeIconLinks(w *strings.Builder, rel string, icons []Icon) {
	for _, ic := range icons {
		w.WriteString(`<link rel="`)
		w.WriteString(html.EscapeString(rel))
		w.WriteString(`" href="`)
		w.WriteString(html.EscapeString(ic.URL))
		w.WriteString(`"`)
		if ic.Sizes != nil {
			w.WriteString(` sizes="`)
			w.WriteString(html.EscapeString(*ic.Sizes))
			w.WriteString(`"`)
		}
		if ic.Type != nil {
			w.WriteString(` type="`)
			w.WriteString(html.EscapeString(*ic.Type))
			w.WriteString(`"`)
		}
		w.WriteString(`/>`)
	}
}
