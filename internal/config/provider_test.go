package config

import "testing"

func TestLookupProviderByDomain(t *testing.T) {
	p := LookupProviderByDomain("qq.com")
	if p == nil || p.ID != "qq" {
		t.Fatalf("qq: %+v", p)
	}
	if p.IMAPHost != "imap.qq.com:993" {
		t.Fatalf("imap host: %s", p.IMAPHost)
	}
}

func TestApplyProviderPreset(t *testing.T) {
	var prof Profile
	prof.User = "me@gmail.com"
	prof.Protocol = "imap"
	got := prof.ApplyProviderPreset()
	if got == nil || got.ID != "gmail" {
		t.Fatalf("preset: %+v", got)
	}
	if prof.IMAP.Host != "imap.gmail.com:993" {
		t.Fatalf("imap: %s", prof.IMAP.Host)
	}
}

func TestApplyProviderPresetMinimalQQ(t *testing.T) {
	var prof Profile
	prof.Protocol = "imap"
	prof.Provider = "qq"
	prof.User = "36291161@qq.com"
	got := prof.ApplyProviderPreset()
	if got == nil || got.ID != "qq" {
		t.Fatalf("preset: %+v", got)
	}
	if prof.IMAP.Host != "imap.qq.com:993" {
		t.Fatalf("imap host: %q", prof.IMAP.Host)
	}
	if prof.SMTP.Host != "smtp.qq.com:465" {
		t.Fatalf("smtp host: %q", prof.SMTP.Host)
	}
	if prof.IMAP.TLS == nil || !*prof.IMAP.TLS {
		t.Fatal("imap tls default")
	}
	if prof.SMTP.TLS == nil || !*prof.SMTP.TLS {
		t.Fatal("smtp tls default for qq:465")
	}
}

func TestApplyProviderPresetExplicitID(t *testing.T) {
	var prof Profile
	prof.Provider = "163"
	prof.User = "x@custom.domain"
	got := prof.ApplyProviderPreset()
	if got == nil || got.ID != "163" {
		t.Fatal("expected 163 preset")
	}
}

func TestShouldNotApplyOnEWS(t *testing.T) {
	var prof Profile
	prof.Protocol = "ews"
	prof.User = "a@qq.com"
	prof.Endpoint = "https://example.com/EWS/Exchange.asmx"
	if prof.ApplyProviderPreset() != nil {
		t.Fatal("should not apply imap preset on ews profile")
	}
}

func TestSentFolderCandidatesQQ(t *testing.T) {
	folders := SentFolderCandidatesForProfile(Profile{User: "a@qq.com", Protocol: "imap"})
	if len(folders) == 0 {
		t.Fatal("expected folders")
	}
	if folders[0] != "Sent Messages" {
		t.Fatalf("qq sent first: %s", folders[0])
	}
}
