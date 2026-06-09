package htmlwriter

import "strings"

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

func IsVoidElement(name string) bool {
	return voidElements[name]
}

func isBoolEnumerableAttribute(name string) bool {
	if name == "draggable" || name == "spellcheck" || name == "contenteditable" {
		return true
	}
	if strings.HasPrefix(name, "aria-") {
		return true
	}
	return false
}

func camelToKebabCase(input string) string {
	var out strings.Builder
	for _, r := range input {
		if r >= 'A' && r <= 'Z' {
			out.WriteByte('-')
			out.WriteRune(r + 32)
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}
