package search

import "testing"

func TestDecodeMIMEHeader(t *testing.T) {
	raw := "=?UTF-8?B?6ZuG6ZuG6ZuG5paH5Lu2?="
	dec := decodeMIMEHeader(raw)
	if dec == raw {
		t.Fatalf("expected decoded subject, got %q", dec)
	}
}

func TestMatchQueryFrom(t *testing.T) {
	row := Row{Subject: "每日信用管家", From: "招商银行信用卡 <notice@cmbchina.com>"}
	opts := Options{Query: "招商银行"}
	if !matchClientFilters(row, nil, opts, nil, "") {
		t.Fatal("query should match from display name")
	}
	opts = Options{Subject: "招商银行"}
	if matchClientFilters(row, nil, opts, nil, "") {
		t.Fatal("subject filter should not match when keyword only in from")
	}
}
