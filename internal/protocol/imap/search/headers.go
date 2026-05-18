package search

import (
	"mime"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var mimeWordDecoder mime.WordDecoder

// decodeMIMEHeader decodes RFC 2047 encoded-words in Subject/From display names.
func decodeMIMEHeader(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if !strings.Contains(s, "=?") {
		return s
	}
	dec, err := mimeWordDecoder.DecodeHeader(s)
	if err != nil {
		return s
	}
	return dec
}

// decodeDisplayName decodes MIME words and GBK raw bytes from some Chinese providers.
func decodeDisplayName(s string) string {
	s = decodeMIMEHeader(s)
	if s == "" {
		return s
	}
	if utf8.ValidString(s) {
		return s
	}
	out, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), s)
	if err == nil && utf8.ValidString(out) {
		return out
	}
	return s
}

// isASCIIOnly reports whether s contains only ASCII (IMAP SEARCH literals for UTF-8
// are poorly supported by QQ/163; non-ASCII filters run client-side only).
func isASCIIOnly(s string) bool {
	for _, r := range s {
		if r >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
