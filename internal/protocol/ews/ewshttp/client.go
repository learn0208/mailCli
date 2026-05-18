// Package ewshttp provides an EWS SOAP client with timeout, retries, and TLS policy.
// SOAP envelope layout derives from github.com/tschuyebuhl/ews (MIT).
package ewshttp

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/go-ntlmssp"
	"github.com/tschuyebuhl/ews"
)

const soapEnd = `
</soap:Body>
</soap:Envelope>`

func soapEnvelopeStart() string {
	ver := strings.TrimSpace(os.Getenv("MAILCLI_SERVER_VERSION"))
	if ver == "" {
		ver = strings.TrimSpace(os.Getenv("EWS_SERVER_VERSION"))
	}
	if ver == "" {
		ver = "Exchange2016"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8" ?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages" xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
<soap:Header>
	<t:RequestServerVersion Version="%s" />
</soap:Header>
<soap:Body>
`, ver)
}

// Client implements ews.Client with timeout and optional NTLM / verbose dump.
type Client struct {
	EWSAddr   string
	Email     string
	Login     ews.LoginStrategy
	Timeout   time.Duration
	Verbose   bool
	NTLM      bool
	RTMutex   *sync.Mutex
	UserAgent string
}

func (c *Client) GetEWSAddr() string  { return c.EWSAddr }
func (c *Client) GetUsername() string { return c.Email }

type mutexRT struct {
	mtx *sync.Mutex
	rt  http.RoundTripper
}

func (m *mutexRT) RoundTrip(r *http.Request) (*http.Response, error) {
	if m.mtx != nil {
		m.mtx.Lock()
		defer m.mtx.Unlock()
	}
	return m.rt.RoundTrip(r)
}

func applyTransport(config *Client) http.RoundTripper {
	godebug := os.Getenv("GODEBUG")
	if !strings.Contains(godebug, "http2client=0") {
		if godebug != "" {
			godebug += ","
		}
		godebug += "http2client=0"
		_ = os.Setenv("GODEBUG", godebug)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if config.NTLM {
		transport.MaxConnsPerHost = 1
		return &mutexRT{mtx: config.RTMutex, rt: ntlmssp.Negotiator{
			RoundTripper: transport,
		}}
	}
	return transport
}

func (c *Client) logRequest(req *http.Request) {
	if !c.Verbose {
		return
	}
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Fprintf(os.Stderr, "EWS request:\n%s\n----\n", string(dump))
}

func (c *Client) logResponse(resp *http.Response) {
	if !c.Verbose {
		return
	}
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Fprintf(os.Stderr, "EWS response:\n%s\n----\n", string(dump))
}

// SendAndReceive posts the inner SOAP body (no envelope) and returns the response body.
func (c *Client) SendAndReceive(body []byte) ([]byte, error) {
	bb := append(append([]byte(soapEnvelopeStart()), body...), []byte(soapEnd)...)

	var lastErr error
	backoff := 100 * time.Millisecond
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		respBytes, err := c.doOnce(bb)
		if err == nil {
			return respBytes, nil
		}
		lastErr = err
		if !isRetriable(err) {
			break
		}
	}
	return nil, lastErr
}

func (c *Client) doOnce(bb []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, c.EWSAddr, bytes.NewReader(bb))
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	c.Login.SetLoginHeaders(req)
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	if ua := strings.TrimSpace(c.UserAgent); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	c.logRequest(req)

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	httpClient := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: applyTransport(c),
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	c.logResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, ews.NewError(resp)
	}
	return io.ReadAll(resp.Body)
}

func isRetriable(err error) bool {
	if err == nil {
		return false
	}
	var se *ews.SoapError
	if errors.As(err, &se) {
		return false
	}
	var he *ews.HTTPError
	if errors.As(err, &he) {
		return he.StatusCode == http.StatusServiceUnavailable || he.StatusCode == http.StatusBadGateway || he.StatusCode == http.StatusGatewayTimeout
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout()
	}
	return false
}

// RequireHTTPS rejects non-https endpoints.
func RequireHTTPS(endpoint string) error {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(endpoint)), "http://") {
		return fmt.Errorf("refuse plaintext HTTP endpoint %q (use https)", endpoint)
	}
	return nil
}

// InsecureTLS returns a transport that skips TLS verify (not used by default).
func InsecureTLS() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return t
}
