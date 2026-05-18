package ews

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tschuyebuhl/ews"

	"github.com/learn0208/mailcli/internal/config"
	"github.com/learn0208/mailcli/internal/protocol/ews/ewshttp"
)

// NewHTTPClient builds an EWS SOAP client from a merged profile.
func NewHTTPClient(p config.Profile, userAgent, password string, verbose bool) (*ewshttp.Client, error) {
	if err := ewshttp.RequireHTTPS(p.Endpoint); err != nil {
		return nil, err
	}
	auth := p.EffectiveAuth()
	var login ews.LoginStrategy
	switch auth {
	case "oauth":
		tok := strings.TrimSpace(p.AccessToken)
		if tok == "" {
			return nil, fmt.Errorf("oauth: set MAILCLI_TOKEN (or EWS_TOKEN) or profile access_token")
		}
		login = ews.XOAuthLogin{Token: tok}
	case "ntlm":
		login = ews.PlainLogin{Username: p.NTLMUsername(), Password: password}
	default:
		login = ews.PlainLogin{Username: strings.TrimSpace(p.User), Password: password}
	}
	var mtx sync.Mutex
	cl := &ewshttp.Client{
		EWSAddr:   strings.TrimSpace(p.Endpoint),
		Email:     strings.TrimSpace(p.User),
		Login:     login,
		Timeout:   parseTimeout(),
		Verbose:   verbose,
		NTLM:      auth == "ntlm",
		UserAgent: strings.TrimSpace(userAgent),
	}
	if cl.NTLM {
		cl.RTMutex = &mtx
	}
	return cl, nil
}

func parseTimeout() time.Duration {
	s := strings.TrimSpace(os.Getenv("MAILCLI_TIMEOUT"))
	if s == "" {
		s = strings.TrimSpace(os.Getenv("EWS_TIMEOUT"))
	}
	if s == "" {
		return 30 * time.Second
	}
	sec, err := strconv.Atoi(s)
	if err != nil || sec <= 0 {
		return 30 * time.Second
	}
	return time.Duration(sec) * time.Second
}

// ValidateProfile checks required EWS connection fields.
func ValidateProfile(p config.Profile) error {
	if strings.TrimSpace(p.Endpoint) == "" {
		return fmt.Errorf("EWS endpoint is required (set endpoint in profile, MAILCLI_ENDPOINT, or --endpoint)")
	}
	if strings.TrimSpace(p.User) == "" {
		return fmt.Errorf("user is required (set user in profile, MAILCLI_USER, or --user)")
	}
	return nil
}
