package config

import "testing"

func TestInferSMTPAddress(t *testing.T) {
	if got := InferSMTPAddress(Profile{User: "a@b.c"}); got != "a@b.c" {
		t.Fatalf("email user: %q", got)
	}
	if got := InferSMTPAddress(Profile{User: "liu.jun", Domain: "zhaopin.com.cn"}); got != "liu.jun@zhaopin.com.cn" {
		t.Fatalf("infer: %q", got)
	}
	if got := InferSMTPAddress(Profile{User: "liu.jun", Domain: "zhaopin.com.cn", SMTPAddress: "x@y.z"}); got != "x@y.z" {
		t.Fatalf("smtp_address override: %q", got)
	}
	if got := InferSMTPAddress(Profile{User: `ZP\liu.jun`}); got != `ZP\liu.jun` {
		t.Fatalf("ntlm user unchanged: %q", got)
	}
}
