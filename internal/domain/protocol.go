package domain

import "strings"

// Protocol identifies the mail access backend for a profile.
type Protocol string

const (
	ProtocolEWS  Protocol = "ews"
	ProtocolIMAP Protocol = "imap" // planned
)

// NormalizeProtocol returns a known protocol or EWS when empty.
func NormalizeProtocol(s string) Protocol {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "ews", "exchange":
		return ProtocolEWS
	case "imap":
		return ProtocolIMAP
	default:
		return Protocol(strings.ToLower(strings.TrimSpace(s)))
	}
}

// Supported reports whether the protocol is implemented.
func (p Protocol) Supported() bool {
	switch p {
	case ProtocolEWS, "":
		return true
	default:
		return false
	}
}
