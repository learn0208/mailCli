package xmlutil

import "strings"

// EscapeAttr escapes text for use in an XML double-quoted attribute value.
func EscapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, "\r", "&#13;")
	s = strings.ReplaceAll(s, "\n", "&#10;")
	return s
}
