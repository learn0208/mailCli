package imap

import (
	"strings"

	"github.com/learn0208/mailcli/internal/config"
)

// ResolveFolder maps common CLI folder names to IMAP mailbox names.
func ResolveFolder(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return "INBOX"
	}
	switch strings.ToLower(n) {
	case "inbox":
		return "INBOX"
	case "sent", "sent items", "sentitems", "sent mail":
		return "Sent"
	case "drafts", "draft":
		return "Drafts"
	case "trash", "deleted", "deleted items":
		return "Trash"
	case "junk", "spam":
		return "Junk"
	default:
		return n
	}
}

// SentFolderCandidates returns mailbox names to try for sent-mail verification.
func SentFolderCandidates(p config.Profile) []string {
	return config.SentFolderCandidatesForProfile(p)
}
