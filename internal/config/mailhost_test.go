package config

import "testing"

func TestHostPort(t *testing.T) {
	h, p, err := HostPort("imap.example.com:993", 143)
	if err != nil || h != "imap.example.com" || p != 993 {
		t.Fatalf("explicit port: %q %d %v", h, p, err)
	}
	h, p, err = HostPort("mail.example.com", 993)
	if err != nil || h != "mail.example.com" || p != 993 {
		t.Fatalf("default port: %q %d %v", h, p, err)
	}
	_, _, err = HostPort("", 993)
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestBoolDefault(t *testing.T) {
	if !BoolDefault(nil, true) {
		t.Fatal("nil should use default true")
	}
	f := false
	if BoolDefault(&f, true) {
		t.Fatal("explicit false")
	}
}
