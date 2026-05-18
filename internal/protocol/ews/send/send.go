package send

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/tschuyebuhl/ews"
)

// Options describes an outbound message.
type Options struct {
	From                     string
	FromDisplayName          string // optional UTF-8 display name (EWS Sender Mailbox Name)
	To                       []string
	Cc                       []string
	Bcc                      []string
	Subject                  string
	TextBody                 string
	HTMLBody                 string
	Attach                   []string
	Importance               string
	FromDisplayNamePlainUTF8 bool // put raw UTF-8 in t:Name instead of RFC 2047 for non-ASCII
	// FromAddressOnly sets t:Name to the same ASCII SMTP as t:EmailAddress and RoutingType SMTP so Exchange
	// is less likely to inject a GBK directory nickname. Requires opts.From to be a full user@domain address.
	FromAddressOnly bool
}

func senderMailbox(email, displayNameUTF8 string, namePlainUTF8, addressOnly bool) (outMailbox, error) {
	mb := outMailbox{EmailAddress: strings.TrimSpace(email)}
	if addressOnly {
		if mb.EmailAddress != "" && isASCIIOnly(mb.EmailAddress) {
			// Use ASCII SMTP as both Name and Address so Exchange is less likely to substitute a GBK GAL nickname in MIME From.
			mb.Name = mb.EmailAddress
			mb.RoutingType = "SMTP"
		}
		return mb, nil
	}
	dn := strings.TrimSpace(displayNameUTF8)
	if dn == "" {
		return mb, nil
	}
	if !utf8.ValidString(dn) {
		return outMailbox{}, fmt.Errorf("sender display name must be valid UTF-8")
	}
	// Already an encoded-word (e.g. from automation); do not double-wrap.
	if strings.HasPrefix(dn, "=?") && strings.Contains(dn, "?=") {
		mb.Name = dn
		return mb, nil
	}
	if namePlainUTF8 || isASCIIOnly(dn) {
		mb.Name = dn
		return mb, nil
	}
	mb.Name = mime.BEncoding.Encode("UTF-8", dn)
	return mb, nil
}

func isASCIIOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// Result is returned after a successful send.
type Result struct {
	ItemID    string `json:"item_id,omitempty"`
	ChangeKey string `json:"change_key,omitempty"`
	// Note is set when the server accepted the send but did not return an ItemId in the SOAP body
	// (common for SendAndSaveCopy on some Exchange builds).
	Note string `json:"note,omitempty"`
	// SentVerified is set when send post-check found a matching copy in Sent Items.
	SentVerified       bool   `json:"sent_verified,omitempty"`
	SentVerifyItemID   string `json:"sent_verify_item_id,omitempty"`
	SentVerifyReceived string `json:"sent_verify_received,omitempty"`
	SentVerifyHint     string `json:"sent_verify_hint,omitempty"`
	VerifyErr          string `json:"verify_err,omitempty"`
}

// Run sends the message via CreateItem (SendAndSaveCopy) and parses ItemId when present.
func Run(c ews.Client, opts Options) (*Result, error) {
	if len(opts.To) == 0 {
		return nil, fmt.Errorf("at least one --to recipient is required")
	}
	if opts.TextBody != "" && opts.HTMLBody != "" {
		return nil, fmt.Errorf("use only one of --text or --html")
	}
	if opts.TextBody == "" && opts.HTMLBody == "" {
		return nil, fmt.Errorf("message body is empty (set --text or --html)")
	}

	isHTML := opts.HTMLBody != ""
	body := opts.TextBody
	if isHTML {
		body = opts.HTMLBody
	}
	bodyType := "Text"
	if isHTML {
		bodyType = "HTML"
	}

	sender := strings.TrimSpace(opts.From)
	if sender == "" {
		sender = c.GetUsername()
	}
	if opts.FromAddressOnly && !strings.Contains(sender, "@") {
		return nil, fmt.Errorf("from_address_only requires a full SMTP address (user@domain); set --from or profile smtp_address / user as email (inferred sender was %q)", sender)
	}
	senderMB, err := senderMailbox(sender, opts.FromDisplayName, opts.FromDisplayNamePlainUTF8, opts.FromAddressOnly)
	if err != nil {
		return nil, err
	}

	atts, err := loadAttachments(opts.Attach)
	if err != nil {
		return nil, err
	}

	msg := outMessage{
		ItemClass: "IPM.Note",
		Subject:   opts.Subject,
		Body: outBody{
			BodyType: bodyType,
			Body:     []byte(body),
		},
		Importance: strings.TrimSpace(opts.Importance),
		Sender: outOneMailbox{
			Mailbox: senderMB,
		},
		ToRecipients:  mailboxes(opts.To),
		CcRecipients:  mailboxesPtr(opts.Cc),
		BccRecipients: mailboxesPtr(opts.Bcc),
		Attachments:   atts,
	}

	item := outCreateItem{
		MessageDisposition: "SendAndSaveCopy",
		SavedItemFolderId: outSavedFolder{
			DistinguishedFolderId: outDistinguishedFolder{ID: "sentitems"},
		},
		Items: outItems{Messages: []outMessage{msg}},
	}

	xmlBytes, err := xml.MarshalIndent(item, "", "  ")
	if err != nil {
		return nil, err
	}
	raw, err := c.SendAndReceive(xmlBytes)
	if err != nil {
		return nil, err
	}
	return parseCreateItemResponse(raw)
}

type outCreateItem struct {
	XMLName            struct{}       `xml:"m:CreateItem"`
	MessageDisposition string         `xml:"MessageDisposition,attr"`
	SavedItemFolderId  outSavedFolder `xml:"m:SavedItemFolderId"`
	Items              outItems       `xml:"m:Items"`
}

type outSavedFolder struct {
	DistinguishedFolderId outDistinguishedFolder `xml:"t:DistinguishedFolderId"`
}

type outDistinguishedFolder struct {
	ID string `xml:"Id,attr"`
}

type outItems struct {
	Messages []outMessage `xml:"t:Message"`
}

type outMessage struct {
	ItemClass     string           `xml:"t:ItemClass"`
	Importance    string           `xml:"t:Importance,omitempty"`
	Subject       string           `xml:"t:Subject"`
	Body          outBody          `xml:"t:Body"`
	Sender        outOneMailbox    `xml:"t:Sender"`
	ToRecipients  outXMailbox      `xml:"t:ToRecipients"`
	CcRecipients  *outXMailbox     `xml:"t:CcRecipients,omitempty"`
	BccRecipients *outXMailbox     `xml:"t:BccRecipients,omitempty"`
	Attachments   *outCreateAttach `xml:"t:Attachments,omitempty"`
}

type outBody struct {
	BodyType string `xml:"BodyType,attr"`
	Body     []byte `xml:",chardata"`
}

type outOneMailbox struct {
	Mailbox outMailbox `xml:"t:Mailbox"`
}

type outMailbox struct {
	Name         string `xml:"t:Name,omitempty"`
	EmailAddress string `xml:"t:EmailAddress"`
	RoutingType  string `xml:"t:RoutingType,omitempty"`
}

type outXMailbox struct {
	Mailbox []outMailbox `xml:"t:Mailbox"`
}

type outCreateAttach struct {
	File []outFileAttachment `xml:"t:FileAttachment"`
}

type outFileAttachment struct {
	Name           string `xml:"t:Name"`
	IsInline       bool   `xml:"t:IsInline"`
	IsContactPhoto bool   `xml:"t:IsContactPhoto"`
	Content        string `xml:"t:Content"`
}

func mailboxes(addrs []string) outXMailbox {
	var xs outXMailbox
	for _, a := range addrs {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		xs.Mailbox = append(xs.Mailbox, outMailbox{EmailAddress: a})
	}
	return xs
}

func mailboxesPtr(addrs []string) *outXMailbox {
	x := mailboxes(addrs)
	if len(x.Mailbox) == 0 {
		return nil
	}
	return &x
}

func loadAttachments(paths []string) (*outCreateAttach, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	var files []outFileAttachment
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read attachment %q: %w", p, err)
		}
		name := filepath.Base(p)
		files = append(files, outFileAttachment{
			Name:           name,
			IsInline:       false,
			IsContactPhoto: false,
			Content:        base64.StdEncoding.EncodeToString(b),
		})
	}
	if len(files) == 0 {
		return nil, nil
	}
	return &outCreateAttach{File: files}, nil
}

type createItemEnvelope struct {
	Body struct {
		CreateItemResponse struct {
			ResponseMessages struct {
				CreateItemResponseMessage struct {
					ResponseClass string `xml:"ResponseClass,attr"`
					MessageText   string `xml:"MessageText"`
					Items         struct {
						Messages []struct {
							ItemID struct {
								Id        string `xml:"Id,attr"`
								ChangeKey string `xml:"ChangeKey,attr"`
							} `xml:"ItemId"`
						} `xml:"Message"`
					} `xml:"Items"`
				} `xml:"CreateItemResponseMessage"`
			} `xml:"ResponseMessages"`
		} `xml:"CreateItemResponse"`
	} `xml:"Body"`
}

func parseCreateItemResponse(bb []byte) (*Result, error) {
	var env createItemEnvelope
	if err := xml.Unmarshal(bb, &env); err != nil {
		return nil, fmt.Errorf("parse create response: %w", err)
	}
	msg := env.Body.CreateItemResponse.ResponseMessages.CreateItemResponseMessage
	if strings.EqualFold(msg.ResponseClass, "Error") {
		return nil, fmt.Errorf("send failed: %s", msg.MessageText)
	}
	if len(msg.Items.Messages) == 0 {
		return &Result{
			Note: noItemIDNote,
		}, nil
	}
	id := msg.Items.Messages[0].ItemID
	return &Result{ItemID: id.Id, ChangeKey: id.ChangeKey}, nil
}

// noItemIDNote explains a successful CreateItem with no returned ItemId (normal for many servers).
const noItemIDNote = "服务器已接受发送，但响应中未包含 ItemId（不少环境下 SendAndSaveCopy 如此）。请到「已发送」文件夹确认邮件是否已保存。"

// SplitAddresses splits comma/semicolon separated lists.
func SplitAddresses(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, sep := range []string{",", ";"} {
		s = strings.ReplaceAll(s, sep, ",")
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// NormalizeImportance maps high/low/normal to EWS literals.
func NormalizeImportance(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return "", nil
	case "high", "urgent":
		return "High", nil
	case "low":
		return "Low", nil
	case "normal", "medium":
		return "Normal", nil
	default:
		return "", fmt.Errorf("unknown importance %q (use high, normal, or low)", s)
	}
}
