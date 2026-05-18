package show

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"

	imapproto "github.com/learn0208/mailcli/internal/protocol/imap"
)

// Options for show command (item_id is UID in the folder).
type Options struct {
	ItemID    string
	ChangeKey string
	Folder    string
	Format    string
}

// Detail is a fetched message.
type Detail struct {
	ItemID           string
	Subject          string
	From             string
	To               []string
	Cc               []string
	Bcc              []string
	DateTimeReceived string
	HasAttachments   bool
	IsRead           bool
	BodyType         string
	Body             string
}

// Run fetches and prints one message by UID.
func Run(c *imapproto.Client, opts Options) error {
	d, err := Fetch(c, opts)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(opts.Format)) {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			ItemID           string   `json:"item_id"`
			Subject          string   `json:"subject"`
			From             string   `json:"from"`
			To               []string `json:"to"`
			Cc               []string `json:"cc,omitempty"`
			Bcc              []string `json:"bcc,omitempty"`
			DateTimeReceived string   `json:"datetime_received"`
			HasAttachments   bool     `json:"has_attachments"`
			IsRead           bool     `json:"is_read"`
			BodyType         string   `json:"body_type"`
			Body             string   `json:"body"`
		}{
			ItemID: d.ItemID, Subject: d.Subject, From: d.From,
			To: d.To, Cc: d.Cc, Bcc: d.Bcc, DateTimeReceived: d.DateTimeReceived,
			HasAttachments: d.HasAttachments, IsRead: d.IsRead,
			BodyType: d.BodyType, Body: d.Body,
		})
	case "html":
		printHeader(d)
		fmt.Fprintln(os.Stdout, "--- body (html) ---")
		fmt.Fprintln(os.Stdout, d.Body)
	default:
		printHeader(d)
		fmt.Fprintln(os.Stdout, "--- body ---")
		fmt.Fprintln(os.Stdout, d.Body)
	}
	return nil
}

// Fetch loads one message by UID.
func Fetch(c *imapproto.Client, opts Options) (*Detail, error) {
	uid, err := strconv.ParseUint(strings.TrimSpace(opts.ItemID), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("item-id must be a numeric UID for IMAP (got %q)", opts.ItemID)
	}
	mbox := imapproto.ResolveFolder(opts.Folder)
	if _, err := c.Select(mbox, false); err != nil {
		return nil, fmt.Errorf("select %q: %w", mbox, err)
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(uid))

	section := &imap.BodySectionName{Peek: true}
	ch := make(chan *imap.Message, 1)
	if err := c.UidFetch(seqset, []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchUid,
		imap.FetchBodyStructure,
		section.FetchItem(),
	}, ch); err != nil {
		return nil, fmt.Errorf("imap uid fetch: %w", err)
	}
	var msg *imap.Message
	for m := range ch {
		msg = m
	}
	if msg == nil {
		return nil, fmt.Errorf("message UID %d not found in %q", uid, mbox)
	}

	d := &Detail{ItemID: fmt.Sprintf("%d", msg.Uid)}
	if msg.Envelope != nil {
		d.Subject = msg.Envelope.Subject
		if len(msg.Envelope.From) > 0 {
			d.From = formatAddress(msg.Envelope.From[0])
		}
		d.To = formatAddresses(msg.Envelope.To)
		d.Cc = formatAddresses(msg.Envelope.Cc)
		d.Bcc = formatAddresses(msg.Envelope.Bcc)
		if !msg.Envelope.Date.IsZero() {
			d.DateTimeReceived = msg.Envelope.Date.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
	}
	if msg.Flags != nil {
		for _, f := range msg.Flags {
			if strings.EqualFold(f, imap.SeenFlag) {
				d.IsRead = true
			}
		}
	}
	if msg.BodyStructure != nil {
		d.HasAttachments = bodyHasAttachments(msg.BodyStructure)
	}

	bodyType, body := extractPreferredBody(msg, section, opts.Format)
	d.BodyType = bodyType
	d.Body = body
	return d, nil
}

func printHeader(d *Detail) {
	fmt.Fprintf(os.Stdout, "Subject:  %s\n", d.Subject)
	fmt.Fprintf(os.Stdout, "From:     %s\n", d.From)
	if len(d.To) > 0 {
		fmt.Fprintf(os.Stdout, "To:       %s\n", strings.Join(d.To, ", "))
	}
	if len(d.Cc) > 0 {
		fmt.Fprintf(os.Stdout, "Cc:       %s\n", strings.Join(d.Cc, ", "))
	}
	if len(d.Bcc) > 0 {
		fmt.Fprintf(os.Stdout, "Bcc:      %s\n", strings.Join(d.Bcc, ", "))
	}
	if d.DateTimeReceived != "" {
		fmt.Fprintf(os.Stdout, "Received: %s\n", d.DateTimeReceived)
	}
	if d.HasAttachments {
		fmt.Fprintln(os.Stdout, "Attach:   yes")
	} else {
		fmt.Fprintln(os.Stdout, "Attach:   no")
	}
	fmt.Fprintf(os.Stdout, "UID:      %s\n", d.ItemID)
}

func extractPreferredBody(msg *imap.Message, section *imap.BodySectionName, format string) (bodyType, body string) {
	r := msg.GetBody(section)
	if r == nil {
		return "text", ""
	}
	ent, err := mail.CreateReader(r)
	if err != nil {
		b, _ := io.ReadAll(io.LimitReader(r, 2<<20))
		return "text", string(b)
	}

	wantHTML := strings.EqualFold(strings.TrimSpace(format), "html")
	var textPlain, textHTML string
	for {
		p, err := ent.NextPart()
		if err != nil {
			break
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := h.ContentType()
			b, _ := io.ReadAll(io.LimitReader(p.Body, 2<<20))
			if strings.Contains(strings.ToLower(ct), "html") {
				textHTML = string(b)
			} else {
				textPlain = string(b)
			}
		case *mail.AttachmentHeader:
			continue
		}
	}
	if wantHTML && textHTML != "" {
		return "html", textHTML
	}
	if textPlain != "" {
		return "text", textPlain
	}
	if textHTML != "" {
		return "html", textHTML
	}
	return "text", ""
}

func bodyHasAttachments(bs *imap.BodyStructure) bool {
	if bs == nil || len(bs.Parts) == 0 {
		return false
	}
	if strings.EqualFold(bs.MIMEType, "multipart") && strings.EqualFold(bs.MIMESubType, "mixed") {
		return len(bs.Parts) > 0
	}
	for _, p := range bs.Parts {
		if p != nil && !strings.EqualFold(p.MIMEType, "text") {
			return true
		}
	}
	return false
}

func formatAddress(addr *imap.Address) string {
	if addr == nil {
		return ""
	}
	if addr.PersonalName != "" && addr.MailboxName != "" && addr.HostName != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName)
	}
	if addr.MailboxName != "" && addr.HostName != "" {
		return addr.MailboxName + "@" + addr.HostName
	}
	return strings.TrimSpace(addr.PersonalName)
}

func formatAddresses(addrs []*imap.Address) []string {
	var out []string
	for _, a := range addrs {
		if s := formatAddress(a); s != "" {
			out = append(out, s)
		}
	}
	return out
}
