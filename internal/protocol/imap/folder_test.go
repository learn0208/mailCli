package imap

import "testing"

func TestResolveFolder(t *testing.T) {
	if got := ResolveFolder("Inbox"); got != "INBOX" {
		t.Fatalf("got %q", got)
	}
	if got := ResolveFolder("sent items"); got != "Sent" {
		t.Fatalf("got %q", got)
	}
	if got := ResolveFolder("Custom"); got != "Custom" {
		t.Fatalf("got %q", got)
	}
}
