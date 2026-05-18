package show

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tschuyebuhl/ews"

	"github.com/learn0208/mailcli/internal/protocol/ews/message"
	"github.com/learn0208/mailcli/internal/protocol/ews/search"
)

// Options for show command.
type Options struct {
	ItemID    string
	ChangeKey string
	Format    string // text | html | json
}

// Run fetches and prints one message by ItemId.
func Run(c ews.Client, opts Options) error {
	d, err := message.Get(c, opts.ItemID, opts.ChangeKey)
	if err != nil {
		return search.ClassifyErr(err)
	}

	switch strings.ToLower(strings.TrimSpace(opts.Format)) {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			ItemID           string                 `json:"item_id"`
			ChangeKey        string                 `json:"change_key,omitempty"`
			Subject          string                 `json:"subject"`
			From             string                 `json:"from"`
			To               []string               `json:"to"`
			Cc               []string               `json:"cc,omitempty"`
			Bcc              []string               `json:"bcc,omitempty"`
			DateTimeReceived string                 `json:"datetime_received"`
			HasAttachments   bool                   `json:"has_attachments"`
			Attachments      []message.AttachmentInfo `json:"attachments,omitempty"`
			IsRead           bool                   `json:"is_read"`
			BodyType         string                 `json:"body_type"`
			Body             string                 `json:"body"`
		}{
			ItemID: d.ItemID, ChangeKey: d.ChangeKey, Subject: d.Subject, From: d.From,
			To: d.To, Cc: d.Cc, Bcc: d.Bcc, DateTimeReceived: d.DateTimeReceived,
			HasAttachments: d.HasAttachments, Attachments: d.Attachments,
			IsRead: d.IsRead, BodyType: d.BodyType, Body: d.Body,
		})
	case "html":
		printHeader(d)
		fmt.Fprintln(os.Stdout, "--- body (html) ---")
		if strings.EqualFold(d.BodyType, "HTML") || looksLikeHTML(d.Body) {
			fmt.Fprintln(os.Stdout, d.Body)
		} else {
			fmt.Fprintln(os.Stdout, d.Body)
		}
	default:
		printHeader(d)
		fmt.Fprintln(os.Stdout, "--- body ---")
		fmt.Fprintln(os.Stdout, d.Body)
	}
	return nil
}

func printHeader(d *message.Detail) {
	fmt.Fprintf(os.Stdout, "Subject:  %s\n", d.Subject)
	fmt.Fprintf(os.Stdout, "From:     %s\n", d.From)
	if len(d.To) > 0 {
		fmt.Fprintf(os.Stdout, "To:       %s\n", message.JoinAddresses(d.To))
	}
	if len(d.Cc) > 0 {
		fmt.Fprintf(os.Stdout, "Cc:       %s\n", message.JoinAddresses(d.Cc))
	}
	if len(d.Bcc) > 0 {
		fmt.Fprintf(os.Stdout, "Bcc:      %s\n", message.JoinAddresses(d.Bcc))
	}
	if d.DateTimeReceived != "" {
		fmt.Fprintf(os.Stdout, "Received: %s\n", d.DateTimeReceived)
	}
	fmt.Fprintf(os.Stdout, "Attach:   %s\n", attachSummary(d))
	if len(d.Attachments) > 0 {
		for _, a := range d.Attachments {
			line := fmt.Sprintf("  - %s", a.Name)
			if a.Size > 0 {
				line += fmt.Sprintf(" (%d bytes)", a.Size)
			}
			if ct := strings.TrimSpace(a.ContentType); ct != "" {
				line += fmt.Sprintf(" [%s]", ct)
			}
			fmt.Fprintln(os.Stdout, line)
		}
	}
	fmt.Fprintf(os.Stdout, "ItemId:   %s\n", d.ItemID)
}

func attachSummary(d *message.Detail) string {
	n := len(d.Attachments)
	if n > 0 {
		if n == 1 {
			return "yes (1 file)"
		}
		return fmt.Sprintf("yes (%d files)", n)
	}
	if d.HasAttachments {
		return "yes (no file attachments listed; may be calendar/inline only)"
	}
	return "no"
}

func looksLikeHTML(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "<html") || strings.Contains(s, "<body") || strings.Contains(s, "<div")
}
