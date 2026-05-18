package domain

import "testing"

func TestNormalizeProtocol(t *testing.T) {
	tests := []struct {
		in   string
		want Protocol
	}{
		{"", ProtocolEWS},
		{"ews", ProtocolEWS},
		{"exchange", ProtocolEWS},
		{"imap", ProtocolIMAP},
		{"unknown", Protocol("unknown")},
	}
	for _, tc := range tests {
		if got := NormalizeProtocol(tc.in); got != tc.want {
			t.Errorf("NormalizeProtocol(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestProtocolSupported(t *testing.T) {
	if !ProtocolEWS.Supported() {
		t.Fatal("ews should be supported")
	}
	if ProtocolIMAP.Supported() {
		t.Fatal("imap not implemented yet")
	}
}
