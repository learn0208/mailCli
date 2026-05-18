package config

import (
	"fmt"
	"strings"
)

// Provider describes a known public IMAP/SMTP mail service preset.
type Provider struct {
	ID          string
	DisplayName string
	Domains     []string
	IMAPHost    string
	SMTPHost    string
	IMAPTLS     bool
	SMTPTLS     bool // implicit TLS (port 465)
	SMTPStartTLS bool
	SentFolders []string
	// AuthHint explains how to obtain credentials (shown by discover / providers).
	AuthHint string
}

var providers = []Provider{
	{
		ID: "gmail", DisplayName: "Gmail",
		Domains: []string{"gmail.com", "googlemail.com"},
		IMAPHost: "imap.gmail.com:993", SMTPHost: "smtp.gmail.com:587",
		IMAPTLS: true, SMTPStartTLS: true,
		SentFolders: []string{"[Gmail]/Sent Mail", "Sent", "Sent Mail"},
		AuthHint: "Enable IMAP in Gmail settings. Use a Google App Password (2FA required) or OAuth; regular password often fails.",
	},
	{
		ID: "yahoo", DisplayName: "Yahoo Mail",
		Domains: []string{"yahoo.com", "yahoo.co.uk", "yahoo.co.jp", "ymail.com", "rocketmail.com"},
		IMAPHost: "imap.mail.yahoo.com:993", SMTPHost: "smtp.mail.yahoo.com:465",
		IMAPTLS: true, SMTPTLS: true,
		SentFolders: []string{"Sent", "Sent Messages"},
		AuthHint: "Generate an app password in Yahoo Account Security; use it as MAILCLI_PASSWORD.",
	},
	{
		ID: "outlook", DisplayName: "Outlook / Hotmail",
		Domains: []string{"outlook.com", "hotmail.com", "live.com", "msn.com"},
		IMAPHost: "outlook.office365.com:993", SMTPHost: "smtp.office365.com:587",
		IMAPTLS: true, SMTPStartTLS: true,
		SentFolders: []string{"Sent Items", "Sent"},
		AuthHint: "Use your Microsoft account password or an app password if 2FA is enabled.",
	},
	{
		ID: "icloud", DisplayName: "iCloud Mail",
		Domains: []string{"icloud.com", "me.com", "mac.com"},
		IMAPHost: "imap.mail.me.com:993", SMTPHost: "smtp.mail.me.com:587",
		IMAPTLS: true, SMTPStartTLS: true,
		SentFolders: []string{"Sent Messages", "Sent"},
		AuthHint: "Create an app-specific password at appleid.apple.com.",
	},
	{
		ID: "qq", DisplayName: "QQ 邮箱 / Foxmail",
		Domains: []string{"qq.com", "foxmail.com"},
		IMAPHost: "imap.qq.com:993", SMTPHost: "smtp.qq.com:465",
		IMAPTLS: true, SMTPTLS: true,
		SentFolders: []string{"Sent Messages", "已发送", "Sent"},
		AuthHint: "QQ Mail does not support QR-code login over IMAP/SMTP. In https://mail.qq.com enable IMAP/SMTP, then create an authorization code (授权码) under Settings → Account. Set MAILCLI_PASSWORD to that code, not your QQ password. Web/App QR login cannot be used by this CLI.",
	},
	{
		ID: "163", DisplayName: "网易 163",
		Domains: []string{"163.com"},
		IMAPHost: "imap.163.com:993", SMTPHost: "smtp.163.com:465",
		IMAPTLS: true, SMTPTLS: true,
		SentFolders: []string{"Sent", "已发送", "Sent Messages"},
		AuthHint: "In 163 web mail enable IMAP/SMTP and create an authorization code (客户端授权密码). Use the code as MAILCLI_PASSWORD, not your login password.",
	},
	{
		ID: "126", DisplayName: "网易 126",
		Domains: []string{"126.com"},
		IMAPHost: "imap.126.com:993", SMTPHost: "smtp.126.com:465",
		IMAPTLS: true, SMTPTLS: true,
		SentFolders: []string{"Sent", "已发送", "Sent Messages"},
		AuthHint: "Same as 163: enable IMAP/SMTP and use the authorization code as MAILCLI_PASSWORD.",
	},
	{
		ID: "yeah", DisplayName: "网易 yeah.net",
		Domains: []string{"yeah.net"},
		IMAPHost: "imap.yeah.net:993", SMTPHost: "smtp.yeah.net:465",
		IMAPTLS: true, SMTPTLS: true,
		SentFolders: []string{"Sent", "已发送"},
		AuthHint: "Enable IMAP/SMTP in web settings; use authorization code as password.",
	},
	{
		ID: "sina", DisplayName: "新浪邮箱",
		Domains: []string{"sina.com", "sina.cn"},
		IMAPHost: "imap.sina.com:993", SMTPHost: "smtp.sina.com:465",
		IMAPTLS: true, SMTPTLS: true,
		SentFolders: []string{"Sent", "已发送"},
		AuthHint: "Enable IMAP in web mail; use authorization/客户端密码 if required.",
	},
	{
		ID: "aliyun", DisplayName: "阿里邮箱",
		Domains: []string{"aliyun.com"},
		IMAPHost: "imap.aliyun.com:993", SMTPHost: "smtp.aliyun.com:465",
		IMAPTLS: true, SMTPTLS: true,
		SentFolders: []string{"Sent Items", "已发送"},
		AuthHint: "Use mailbox password or client authorization per Aliyun mail help.",
	},
}

// AllProviders returns built-in provider presets.
func AllProviders() []Provider {
	out := make([]Provider, len(providers))
	copy(out, providers)
	return out
}

// LookupProvider finds a preset by explicit profile provider id or email domain.
func LookupProvider(p Profile) *Provider {
	if id := strings.ToLower(strings.TrimSpace(p.Provider)); id != "" {
		for i := range providers {
			if providers[i].ID == id {
				cp := providers[i]
				return &cp
			}
		}
	}
	domain := emailDomain(p.User)
	if domain == "" {
		return nil
	}
	for i := range providers {
		for _, d := range providers[i].Domains {
			if domain == d {
				cp := providers[i]
				return &cp
			}
		}
	}
	return nil
}

// LookupProviderByDomain returns a preset for an email domain (e.g. qq.com).
func LookupProviderByDomain(domain string) *Provider {
	domain = strings.ToLower(strings.TrimSpace(domain))
	for i := range providers {
		for _, d := range providers[i].Domains {
			if domain == d {
				cp := providers[i]
				return &cp
			}
		}
	}
	return nil
}

// LookupProviderByID returns a preset by id (e.g. qq, gmail).
func LookupProviderByID(id string) *Provider {
	p := Profile{Provider: id}
	return LookupProvider(p)
}

func emailDomain(addr string) string {
	addr = strings.ToLower(strings.TrimSpace(addr))
	at := strings.LastIndex(addr, "@")
	if at < 0 {
		return ""
	}
	return strings.TrimSpace(addr[at+1:])
}

// ApplyProviderPreset fills IMAP/SMTP hosts and TLS defaults from a known provider.
// It may set protocol to imap when appropriate. Returns the matched preset or nil.
func (p *Profile) ApplyProviderPreset() *Provider {
	if !shouldApplyIMAPProvider(*p) {
		return nil
	}
	prov := LookupProvider(*p)
	if prov == nil {
		return nil
	}
	if strings.TrimSpace(p.Protocol) == "" && strings.TrimSpace(p.Endpoint) == "" {
		p.Protocol = "imap"
	}
	if strings.TrimSpace(p.IMAP.Host) == "" {
		p.IMAP.Host = prov.IMAPHost
	}
	if strings.TrimSpace(p.SMTP.Host) == "" {
		p.SMTP.Host = prov.SMTPHost
	}
	if p.IMAP.TLS == nil {
		v := prov.IMAPTLS
		p.IMAP.TLS = &v
	}
	if p.SMTP.TLS == nil && prov.SMTPTLS {
		v := true
		p.SMTP.TLS = &v
	}
	if p.SMTP.StartTLS == nil && prov.SMTPStartTLS {
		v := true
		p.SMTP.StartTLS = &v
	}
	return prov
}

func shouldApplyIMAPProvider(p Profile) bool {
	proto := strings.ToLower(strings.TrimSpace(p.Protocol))
	if proto == "ews" || proto == "exchange" {
		return false
	}
	if proto == "imap" || strings.TrimSpace(p.Provider) != "" {
		return true
	}
	if strings.TrimSpace(p.Endpoint) != "" {
		return false
	}
	return LookupProvider(p) != nil
}

// SentFolderCandidatesForProfile returns sent-folder names to try for sent-mail verification.
func SentFolderCandidatesForProfile(p Profile) []string {
	base := []string{"Sent", "Sent Items", "Sent Mail", "Sent Messages", "[Gmail]/Sent Mail", "INBOX.Sent", "已发送"}
	if prov := LookupProvider(p); prov != nil && len(prov.SentFolders) > 0 {
		seen := map[string]struct{}{}
		var out []string
		for _, f := range append(prov.SentFolders, base...) {
			if _, ok := seen[f]; ok {
				continue
			}
			seen[f] = struct{}{}
			out = append(out, f)
		}
		return out
	}
	return base
}

// FormatProviderYAML returns an example profile snippet for a provider and user email.
func FormatProviderYAML(prov Provider, email string) string {
	return fmt.Sprintf(`protocol: imap
provider: %s
user: %s
imap:
  host: %s
smtp:
  host: %s
# MAILCLI_PASSWORD=<see: mailcli providers show %s>
`, prov.ID, email, prov.IMAPHost, prov.SMTPHost, prov.ID)
}
