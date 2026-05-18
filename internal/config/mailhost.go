package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// IMAPSettings holds IMAP connection options for a profile.
type IMAPSettings struct {
	Host               string `mapstructure:"host"`
	TLS                *bool  `mapstructure:"tls"`
	StartTLS           *bool  `mapstructure:"starttls"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

// SMTPSettings holds SMTP connection options for a profile.
type SMTPSettings struct {
	Host               string `mapstructure:"host"`
	TLS                *bool  `mapstructure:"tls"`
	StartTLS           *bool  `mapstructure:"starttls"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

// HostPort splits host[:port] and returns defaults when port is omitted.
func HostPort(host string, defaultPort int) (string, int, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", 0, fmt.Errorf("host is required")
	}
	if strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			return "", 0, fmt.Errorf("invalid host %q: %w", host, err)
		}
		port, err := strconv.Atoi(p)
		if err != nil || port <= 0 {
			return "", 0, fmt.Errorf("invalid port in %q", host)
		}
		return h, port, nil
	}
	return host, defaultPort, nil
}

// BoolDefault returns *b when set, otherwise def.
func BoolDefault(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}
