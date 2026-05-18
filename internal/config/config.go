package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Profile holds merged connection settings for one account.
type Profile struct {
	// Protocol is the mail backend: ews (default), imap.
	Protocol string `mapstructure:"protocol"`
	// Provider selects a built-in preset (gmail, qq, 163, yahoo, …) or is inferred from user@domain.
	Provider string `mapstructure:"provider"`
	IMAP     IMAPSettings `mapstructure:"imap"`
	SMTP        SMTPSettings `mapstructure:"smtp"`
	Endpoint    string `mapstructure:"endpoint"`
	User        string `mapstructure:"user"`
	AuthType    string `mapstructure:"auth_type"`
	Domain      string `mapstructure:"domain"`
	Password    string `mapstructure:"password"`
	AccessToken string `mapstructure:"access_token"`
	// SMTPAddress optional explicit default SMTP / From address when `user` is not an email (e.g. liu.jun + domain inference is wrong).
	SMTPAddress string `mapstructure:"smtp_address"`
	// DisplayName is optional UTF-8 sender display name for outgoing mail (EWS t:Sender/t:Mailbox/t:Name).
	// When set, clients are less likely to show mojibake than names taken only from the directory (e.g. GBK/Latin-1 mismatch).
	DisplayName string `mapstructure:"display_name"`
	// FromDisplayNamePlainUTF8 disables RFC2047 wrapping for display_name / --from-name (default is to use RFC2047 for non-ASCII).
	FromDisplayNamePlainUTF8 bool `mapstructure:"from_display_name_plain_utf8"`
	// FromAddressOnly avoids AD/GAL GBK nicknames in MIME: use SMTP address as both t:EmailAddress and t:Name (ASCII),
	// plus RoutingType SMTP. Omit display_name; infer full address via smtp_address or user@domain when --from is omitted.
	FromAddressOnly bool `mapstructure:"from_address_only"`
}

// AppSettings holds top-level YAML options (not tied to a single profile).
type AppSettings struct {
	UserAgent string `mapstructure:"user_agent"`
}

// Load reads the config file (if present) and returns the named profile and file-level app settings.
// Call ApplyEnv / ApplyEnvAppSettings after CLI overrides as needed.
func Load(configPath, profileName string) (Profile, AppSettings, error) {
	var root struct {
		UserAgent string             `mapstructure:"user_agent"`
		Profiles  map[string]Profile `mapstructure:"profiles"`
	}

	if strings.TrimSpace(configPath) != "" {
		v := viper.New()
		switch strings.TrimPrefix(strings.ToLower(filepath.Ext(configPath)), ".") {
		case "yaml", "yml":
			v.SetConfigType("yaml")
		case "toml":
			v.SetConfigType("toml")
		default:
			v.SetConfigType("yaml")
		}
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			if !errors.Is(err, os.ErrNotExist) && !isNotExist(err) {
				return Profile{}, AppSettings{}, fmt.Errorf("read config: %w", err)
			}
		} else if err := v.Unmarshal(&root); err != nil {
			return Profile{}, AppSettings{}, fmt.Errorf("parse config: %w", err)
		}
	}

	app := AppSettings{UserAgent: strings.TrimSpace(root.UserAgent)}

	if root.Profiles == nil {
		root.Profiles = map[string]Profile{}
	}
	p, ok := root.Profiles[profileName]
	if !ok && profileName != "" && len(root.Profiles) > 0 {
		return Profile{}, app, fmt.Errorf("profile %q not found in %s", profileName, configPath)
	}

	return p, app, nil
}

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	ls := strings.ToLower(err.Error())
	return strings.Contains(ls, "cannot find the file") ||
		strings.Contains(ls, "cannot find the path") ||
		strings.Contains(ls, "no such file")
}

func envFirst(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// ApplyEnv applies environment variables (highest precedence: env over file and CLI).
// MAILCLI_* is preferred; EWS_* is accepted for backward compatibility with early builds.
func ApplyEnv(p *Profile) {
	if v := envFirst("MAILCLI_PROTOCOL", "EWS_PROTOCOL"); v != "" {
		p.Protocol = v
	}
	if v := envFirst("MAILCLI_ENDPOINT", "EWS_ENDPOINT"); v != "" {
		p.Endpoint = v
	}
	if v := envFirst("MAILCLI_USER", "EWS_USER"); v != "" {
		p.User = v
	}
	if v := envFirst("MAILCLI_PASSWORD", "EWS_PASSWORD"); v != "" {
		p.Password = v
	}
	if v := envFirst("MAILCLI_TOKEN", "EWS_TOKEN"); v != "" {
		p.AccessToken = v
	}
	if v := envFirst("MAILCLI_AUTH_TYPE", "EWS_AUTH_TYPE"); v != "" {
		p.AuthType = v
	}
	if v := envFirst("MAILCLI_DOMAIN", "EWS_DOMAIN"); v != "" {
		p.Domain = v
	}
	if v := envFirst("MAILCLI_SMTP_ADDRESS", "EWS_SMTP_ADDRESS"); v != "" {
		p.SMTPAddress = v
	}
	if v := envFirst("MAILCLI_IMAP_HOST"); v != "" {
		p.IMAP.Host = v
	}
	if v := envFirst("MAILCLI_SMTP_HOST"); v != "" {
		p.SMTP.Host = v
	}
	if v := envFirst("MAILCLI_PROVIDER"); v != "" {
		p.Provider = v
	}
}

// ApplyEnvAppSettings applies global env overrides for app-level settings.
func ApplyEnvAppSettings(app *AppSettings) {
	if v := envFirst("MAILCLI_USER_AGENT", "EWS_USER_AGENT"); v != "" {
		app.UserAgent = v
	}
}

// EffectiveAuth returns normalized auth type: basic | ntlm | oauth.
func (p Profile) EffectiveAuth() string {
	a := strings.ToLower(strings.TrimSpace(p.AuthType))
	if a == "" {
		return "basic"
	}
	return a
}

// NTLMUsername returns DOMAIN\user when domain is set (NTLM), else user.
func (p Profile) NTLMUsername() string {
	d := strings.TrimSpace(p.Domain)
	u := strings.TrimSpace(p.User)
	if d == "" {
		return u
	}
	if strings.Contains(u, `\`) {
		return u
	}
	return d + `\` + u
}

// InferSMTPAddress returns a best-effort SMTP address for CreateItem From/Sender when --from is omitted.
// If user already looks like an email, it is returned; otherwise user@domain is built from profile.Domain.
func InferSMTPAddress(p Profile) string {
	if s := strings.TrimSpace(p.SMTPAddress); s != "" {
		return s
	}
	u := strings.TrimSpace(p.User)
	if u == "" {
		return ""
	}
	if strings.Contains(u, "@") {
		return u
	}
	if strings.Contains(u, `\`) {
		return u
	}
	d := strings.TrimSpace(p.Domain)
	if d == "" {
		return u
	}
	return u + "@" + d
}
