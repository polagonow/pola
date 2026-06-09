package htmlwriter

import (
	"fmt"
	"html"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Writer struct {
	w io.Writer
}

func New(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) WriteRaw(s string) (int, error) {
	return io.WriteString(w.w, s)
}

func (w *Writer) WriteText(s string) (int, error) {
	return io.WriteString(w.w, html.EscapeString(s))
}

func (w *Writer) WriteAttr(name string, value any) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case bool:
		if isBoolEnumerableAttribute(name) {
			_, err := fmt.Fprintf(w.w, ` %s="%s"`, name, strconv.FormatBool(v))
			return err
		}
		if v {
			_, err := fmt.Fprintf(w.w, ` %s`, name)
			return err
		}
		return nil

	case map[string]any:
		if name == "style" {
			return w.writeStyleAttr(v)
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := w.WriteAttr(k, v[k]); err != nil {
				return err
			}
		}
		return nil

	case []any:
		if name == "class" {
			return w.writeClassAttr(v)
		}
		parts := make([]string, 0, len(v))
		for _, item := range v {
			s, err := stringValue(item)
			if err != nil {
				return err
			}
			parts = append(parts, s)
		}
		_, err := fmt.Fprintf(w.w, ` %s="%s"`, name, strings.Join(parts, ","))
		return err

	case string:
		_, err := fmt.Fprintf(w.w, ` %s="%s"`, name, html.EscapeString(v))
		return err

	default:
		s, err := stringValue(v)
		if err != nil {
			return fmt.Errorf("htmlwriter: unsupported attr type %T for %q", value, name)
		}
		_, err = fmt.Fprintf(w.w, ` %s="%s"`, name, s)
		return err
	}
}

func (w *Writer) writeStyleAttr(styles map[string]any) error {
	if len(styles) == 0 {
		return nil
	}
	keys := make([]string, 0, len(styles))
	for k := range styles {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(styles))
	for _, k := range keys {
		v, err := rawStringValue(styles[k])
		if err != nil {
			return err
		}
		v = strings.ReplaceAll(v, `;`, `\;`)
		v = strings.ReplaceAll(v, `"`, `&quot;`)
		parts = append(parts, camelToKebabCase(k)+":"+v)
	}
	_, err := fmt.Fprintf(w.w, ` style="%s"`, strings.Join(parts, ";"))
	return err
}

func (w *Writer) writeClassAttr(items []any) error {
	classes := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil || item == false {
			continue
		}
		switch v := item.(type) {
		case bool:
			if !v {
				continue
			}
		case int:
			if v == 0 {
				continue
			}
			classes = append(classes, strconv.Itoa(v))
			continue
		case float64:
			if v == 0 {
				continue
			}
			classes = append(classes, strconv.FormatFloat(v, 'f', -1, 64))
			continue
		}
		s, err := stringValue(item)
		if err != nil {
			return err
		}
		if s != "" {
			classes = append(classes, s)
		}
	}
	if len(classes) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(w.w, ` class="%s"`, strings.Join(classes, " "))
	return err
}

func (w *Writer) WriteAttrs(attrs map[string]any) error {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		if k == "children" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := w.WriteAttr(k, attrs[k]); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) OpenTag(tag string, attrs map[string]any) error {
	if _, err := fmt.Fprintf(w.w, "<%s", tag); err != nil {
		return err
	}
	if err := w.WriteAttrs(attrs); err != nil {
		return err
	}
	_, err := io.WriteString(w.w, ">")
	return err
}

func (w *Writer) VoidTag(tag string, attrs map[string]any) error {
	if _, err := fmt.Fprintf(w.w, "<%s", tag); err != nil {
		return err
	}
	if err := w.WriteAttrs(attrs); err != nil {
		return err
	}
	_, err := io.WriteString(w.w, " />")
	return err
}

func (w *Writer) CloseTag(tag string) error {
	_, err := fmt.Fprintf(w.w, "</%s>", tag)
	return err
}

func rawStringValue(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case fmt.Stringer:
		return val.String(), nil
	default:
		return stringValue(v)
	}
}

func stringValue(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return html.EscapeString(val), nil
	case time.Time:
		return val.UTC().Format(time.RFC3339), nil
	case int:
		return strconv.Itoa(val), nil
	case int8:
		return strconv.FormatInt(int64(val), 10), nil
	case int16:
		return strconv.FormatInt(int64(val), 10), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case uint:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint64:
		return strconv.FormatUint(val, 10), nil
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32), nil
	case float64:
		if math.IsInf(val, 1) {
			return "Infinity", nil
		}
		if math.IsInf(val, -1) {
			return "-Infinity", nil
		}
		if math.IsNaN(val) {
			return "NaN", nil
		}
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case fmt.Stringer:
		return html.EscapeString(val.String()), nil
	default:
		return "", fmt.Errorf("htmlwriter: unsupported value type %T", v)
	}
}
