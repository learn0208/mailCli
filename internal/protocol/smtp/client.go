package smtp

import (
	"fmt"
	"strings"

	"github.com/learn0208/mailcli/internal/config"
)

// ValidateProfile checks SMTP profile fields.
func ValidateProfile(p config.Profile) error {
	if strings.TrimSpace(p.User) == "" {
		return fmt.Errorf("user is required (set user in profile, MAILCLI_USER, or --user)")
	}
	if strings.TrimSpace(p.SMTP.Host) == "" {
		return fmt.Errorf("smtp.host is required (profile smtp.host or MAILCLI_SMTP_HOST)")
	}
	return nil
}

// Endpoint returns host, port, useTLS, useStartTLS for the profile.
func Endpoint(p config.Profile) (host string, port int, useTLS, useStartTLS bool, err error) {
	host, port, err = config.HostPort(p.SMTP.Host, 587)
	if err != nil {
		return "", 0, false, false, err
	}
	useTLS = config.BoolDefault(p.SMTP.TLS, port == 465)
	useStartTLS = config.BoolDefault(p.SMTP.StartTLS, port == 587 || port == 25)
	if port == 465 {
		useStartTLS = false
	}
	return host, port, useTLS, useStartTLS, nil
}

// AuthUser returns the SMTP AUTH username.
func AuthUser(p config.Profile) string {
	return strings.TrimSpace(p.User)
}

// TLSInsecure reports whether TLS certificate verification is skipped.
func TLSInsecure(p config.Profile) bool {
	return p.SMTP.InsecureSkipVerify
}

// ServerName returns the TLS ServerName (SNI).
func ServerName(host string) string {
	return host
}

// Addr formats host:port for net.Dial.
func Addr(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}
