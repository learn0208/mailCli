package imap

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"

	imapclient "github.com/emersion/go-imap/client"

	"github.com/learn0208/mailcli/internal/config"
)

// Client wraps a logged-in IMAP session.
type Client struct {
	*imapclient.Client
}

// ValidateProfile checks IMAP profile fields.
func ValidateProfile(p config.Profile) error {
	if strings.TrimSpace(p.User) == "" {
		return fmt.Errorf("user is required (set user in profile, MAILCLI_USER, or --user)")
	}
	if strings.TrimSpace(p.IMAP.Host) == "" {
		return fmt.Errorf("imap.host is required (set imap.host, MAILCLI_IMAP_HOST, provider preset, or run: mailcli providers list)")
	}
	return nil
}

// Connect dials and logs in to the configured IMAP server.
func Connect(p config.Profile, password string) (*Client, error) {
	host, port, err := config.HostPort(p.IMAP.Host, 993)
	if err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: p.IMAP.InsecureSkipVerify,
	}

	useTLS := config.BoolDefault(p.IMAP.TLS, port == 993)
	useStartTLS := config.BoolDefault(p.IMAP.StartTLS, port == 143)

	var c *imapclient.Client
	switch {
	case useTLS && port != 143:
		c, err = imapclient.DialTLS(addr, tlsConfig)
	default:
		c, err = imapclient.Dial(addr)
		if err == nil && useStartTLS {
			if err = c.StartTLS(tlsConfig); err != nil {
				_ = c.Logout()
				return nil, fmt.Errorf("imap STARTTLS: %w", err)
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("imap connect %s: %w", addr, err)
	}
	if err := c.Login(strings.TrimSpace(p.User), password); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("imap login: %w", err)
	}
	return &Client{Client: c}, nil
}

// Close logs out when the client is non-nil.
func (c *Client) Close() {
	if c == nil || c.Client == nil {
		return
	}
	_ = c.Logout()
}
