package send

import (
	"strings"
	"testing"
)

func TestSenderMailbox_rfc2047NonASCII(t *testing.T) {
	mb, err := senderMailbox("a@b.c", "刘军", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if mb.EmailAddress != "a@b.c" {
		t.Fatalf("email: %q", mb.EmailAddress)
	}
	if !strings.HasPrefix(mb.Name, "=?UTF-8?") || !strings.Contains(mb.Name, "?=") {
		t.Fatalf("expected RFC2047 UTF-8 encoded-word, got %q", mb.Name)
	}
}

func TestSenderMailbox_plainUTF8Flag(t *testing.T) {
	mb, err := senderMailbox("a@b.c", "刘军", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if mb.Name != "刘军" {
		t.Fatalf("got %q", mb.Name)
	}
}

func TestSenderMailbox_asciiUnencoded(t *testing.T) {
	mb, err := senderMailbox("a@b.c", "Bob", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if mb.Name != "Bob" {
		t.Fatalf("got %q", mb.Name)
	}
}

func TestSenderMailbox_addressOnlyDupesSMTPName(t *testing.T) {
	mb, err := senderMailbox("liu.jun@zhaopin.com.cn", "刘军", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if mb.EmailAddress != "liu.jun@zhaopin.com.cn" || mb.Name != mb.EmailAddress {
		t.Fatalf("want Name==Email, got name=%q email=%q", mb.Name, mb.EmailAddress)
	}
	if mb.RoutingType != "SMTP" {
		t.Fatalf("routing: %q", mb.RoutingType)
	}
}
