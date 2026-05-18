package send

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	netmail "net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"

	"github.com/learn0208/mailcli/internal/config"
	imapproto "github.com/learn0208/mailcli/internal/protocol/imap"
	imapsearch "github.com/learn0208/mailcli/internal/protocol/imap/search"
	smtpproto "github.com/learn0208/mailcli/internal/protocol/smtp"
)

// Options describes an outbound message.
type Options struct {
	From            string
	FromDisplayName string
	To              []string
	Cc              []string
	Bcc             []string
	Subject         string
	TextBody        string
	HTMLBody        string
	Attach          []string
}

// Result is returned after a successful send.
type Result struct {
	Note               string `json:"note,omitempty"`
	SentVerified       bool   `json:"sent_verified,omitempty"`
	SentVerifyItemID   string `json:"sent_verify_item_id,omitempty"`
	SentVerifyReceived string `json:"sent_verify_received,omitempty"`
	SentVerifyHint     string `json:"sent_verify_hint,omitempty"`
	VerifyErr          string `json:"verify_err,omitempty"`
}

// Run sends the message via SMTP.
func Run(p config.Profile, password string, opts Options) (*Result, error) {
	if len(opts.To) == 0 {
		return nil, fmt.Errorf("at least one --to recipient is required")
	}
	if opts.TextBody != "" && opts.HTMLBody != "" {
		return nil, fmt.Errorf("use only one of --text or --html")
	}
	if opts.TextBody == "" && opts.HTMLBody == "" {
		return nil, fmt.Errorf("message body is empty (set --text or --html)")
	}
	from := strings.TrimSpace(opts.From)
	if from == "" {
		from = config.InferSMTPAddress(p)
	}
	if from == "" {
		return nil, fmt.Errorf("sender address is required (--from or profile user/smtp_address)")
	}

	raw, err := buildMessage(from, opts)
	if err != nil {
		return nil, err
	}
	if err := deliver(p, password, from, allRecipients(opts), raw); err != nil {
		return nil, err
	}
	return &Result{Note: "message accepted by SMTP server"}, nil
}

// VerifySent searches common Sent folders via IMAP for a matching copy.
func VerifySent(p config.Profile, password string, subject string, to []string, wait time.Duration) (*Result, error) {
	if wait > 0 {
		time.Sleep(wait)
	}
	ic, err := imapproto.Connect(p, password)
	if err != nil {
		return &Result{VerifyErr: err.Error()}, nil
	}
	defer ic.Close()

	ref := time.Now()
	since := ref.Add(-30 * time.Minute).Format(time.RFC3339)
	for _, folder := range imapproto.SentFolderCandidates(p) {
		rows, err := imapsearch.Find(ic, imapsearch.Options{
			Folder: folder,
			Subject: subject,
			Since:  since,
			Limit:  20,
		})
		if err != nil {
			continue
		}
		for _, row := range rows {
			if matchSent(row, subject, to) {
				return &Result{
					SentVerified:       true,
					SentVerifyItemID:   row.ItemID,
					SentVerifyReceived: row.DateTimeReceived,
				}, nil
			}
		}
	}
	return &Result{
		SentVerifyHint: "no matching message found in Sent folders (tried common names)",
	}, nil
}

func matchSent(row imapsearch.Row, subject string, to []string) bool {
	if !strings.EqualFold(strings.TrimSpace(row.Subject), strings.TrimSpace(subject)) {
		return false
	}
	if len(to) == 0 {
		return true
	}
	combined := strings.ToLower(strings.Join(append(row.To, row.Cc...), " "))
	for _, addr := range to {
		if !strings.Contains(combined, strings.ToLower(strings.TrimSpace(addr))) {
			return false
		}
	}
	return true
}

func allRecipients(opts Options) []string {
	var out []string
	out = append(out, opts.To...)
	out = append(out, opts.Cc...)
	out = append(out, opts.Bcc...)
	return out
}

func buildMessage(from string, opts Options) ([]byte, error) {
	header := mail.Header{}
	fromAddr := formatFrom(from, opts.FromDisplayName)
	header.Set("From", fromAddr)
	header.Set("To", strings.Join(opts.To, ", "))
	if len(opts.Cc) > 0 {
		header.Set("Cc", strings.Join(opts.Cc, ", "))
	}
	if len(opts.Bcc) > 0 {
		header.Set("Bcc", strings.Join(opts.Bcc, ", "))
	}
	header.Set("Subject", opts.Subject)
	header.Set("Date", time.Now().Format(time.RFC1123Z))
	header.Set("MIME-Version", "1.0")

	var buf bytes.Buffer
	if len(opts.Attach) == 0 {
		if opts.HTMLBody != "" {
			header.Set("Content-Type", "text/html; charset=UTF-8")
		} else {
			header.Set("Content-Type", "text/plain; charset=UTF-8")
		}
		w, err := mail.CreateSingleInlineWriter(&buf, header)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(w, strings.NewReader(bodyContent(opts))); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	mw, err := mail.CreateWriter(&buf, header)
	if err != nil {
		return nil, err
	}

	ih := mail.InlineHeader{}
	if opts.HTMLBody != "" {
		ih.Set("Content-Type", "text/html; charset=UTF-8")
	} else {
		ih.Set("Content-Type", "text/plain; charset=UTF-8")
	}
	w, err := mw.CreateSingleInline(ih)
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(w, bodyContent(opts)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	for _, path := range opts.Attach {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read attachment %s: %w", path, err)
		}
		name := filepath.Base(path)
		ah := mail.AttachmentHeader{}
		ct := mime.TypeByExtension(filepath.Ext(path))
		if ct == "" {
			ct = "application/octet-stream"
		}
		ah.Set("Content-Type", ct)
		ah.SetFilename(name)
		aw, err := mw.CreateAttachment(ah)
		if err != nil {
			return nil, err
		}
		if _, err := aw.Write(data); err != nil {
			return nil, err
		}
		if err := aw.Close(); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func bodyContent(opts Options) string {
	if opts.HTMLBody != "" {
		return opts.HTMLBody
	}
	return opts.TextBody
}

func formatFrom(addr, displayName string) string {
	displayName = strings.TrimSpace(displayName)
	addr = strings.TrimSpace(addr)
	if displayName == "" {
		return addr
	}
	return (&netmail.Address{Name: displayName, Address: addr}).String()
}

func deliver(p config.Profile, password string, from string, to []string, raw []byte) error {
	host, port, useTLS, useStartTLS, err := smtpproto.Endpoint(p)
	if err != nil {
		return err
	}
	addr := smtpproto.Addr(host, port)
	tlsConfig := &tls.Config{
		ServerName:         smtpproto.ServerName(host),
		InsecureSkipVerify: smtpproto.TLSInsecure(p),
	}

	var conn net.Conn
	if useTLS {
		conn, err = tls.Dial("tcp", addr, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp connect %s: %w", addr, err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = c.Quit() }()

	if useStartTLS {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("smtp STARTTLS: %w", err)
			}
		}
	}

	authUser := smtpproto.AuthUser(p)
	if ok, _ := c.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", authUser, password, host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(strings.TrimSpace(rcpt)); err != nil {
			return fmt.Errorf("smtp RCPT TO %s: %w", rcpt, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return nil
}

// SplitAddresses splits comma-separated addresses (shared with EWS send).
func SplitAddresses(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
