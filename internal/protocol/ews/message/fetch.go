package message

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/tschuyebuhl/ews"
)

// Detail is a mail message returned by GetItem.
type Detail struct {
	ItemID           string
	ChangeKey        string
	Subject          string
	From             string
	To               []string
	Cc               []string
	Bcc              []string
	DateTimeReceived string
	HasAttachments   bool
	Attachments      []AttachmentInfo
	IsRead           bool
	BodyType         string
	Body             string
}

// AttachmentInfo is metadata for one file attachment (no binary content).
type AttachmentInfo struct {
	Name        string `json:"name"`
	Size        uint64 `json:"size,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// Get loads one message by ItemId (retries without ChangeKey on failure).
func Get(c ews.Client, id, changeKey string) (*Detail, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("item id is required")
	}
	changeKey = strings.TrimSpace(changeKey)
	var tries []ews.ItemId
	if changeKey != "" {
		tries = append(tries, ews.ItemId{Id: id, ChangeKey: changeKey})
	}
	tries = append(tries, ews.ItemId{Id: id})
	var lastErr error
	for _, itemID := range tries {
		details, err := getBatch(c, []ews.GetItemMessage{{ItemId: itemID}}, ews.BaseShapeAllProperties)
		if err != nil {
			lastErr = err
			continue
		}
		if len(details) == 0 {
			lastErr = fmt.Errorf("empty response")
			continue
		}
		return &details[0], nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("not found")
	}
	return nil, lastErr
}

// GetBatch loads multiple messages in one GetItem call.
func GetBatch(c ews.Client, ids []ews.ItemId) ([]Detail, error) {
	items := make([]ews.GetItemMessage, 0, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id.Id) == "" {
			continue
		}
		items = append(items, ews.GetItemMessage{ItemId: id})
	}
	return getBatch(c, items, ews.BaseShapeDefault)
}

func getBatch(c ews.Client, items []ews.GetItemMessage, baseShape ews.BaseShape) ([]Detail, error) {
	if len(items) == 0 {
		return nil, nil
	}
	body, err := MarshalGetItemRequest(items, baseShape)
	if err != nil {
		return nil, err
	}
	raw, err := c.SendAndReceive(body)
	if err != nil {
		return nil, err
	}
	return parseGetItemResponse(raw)
}

// MarshalGetItemRequest builds the inner SOAP body for GetItem. Empty ChangeKey is omitted from XML
// because Exchange rejects ChangeKey="" with ErrorInvalidChangeKey.
// Use BaseShapeDefault for batch/list enrichment; BaseShapeAllProperties when you need attachment metadata
// (Default shape returns an empty attachment collection for email messages per Exchange behavior).
func MarshalGetItemRequest(items []ews.GetItemMessage, baseShape ews.BaseShape) ([]byte, error) {
	if baseShape == "" {
		baseShape = ews.BaseShapeDefault
	}
	req := &ews.GetItemRequest{
		ItemShape: ews.GetItemShape{
			BaseShape:          baseShape,
			IncludeMimeContent: ews.BooleanType(false),
		},
		Items: items,
	}
	body, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, err
	}
	s := string(body)
	// encoding/xml emits ChangeKey="" when empty; Exchange treats that as an invalid key.
	s = strings.ReplaceAll(s, ` ChangeKey=""`, "")
	return []byte(s), nil
}

type getItemEnvelope struct {
	Body struct {
		GetItemResponse struct {
			ResponseMessages struct {
				Messages []getItemResponseMessage `xml:"GetItemResponseMessage"`
			} `xml:"ResponseMessages"`
		} `xml:"GetItemResponse"`
	} `xml:"Body"`
}

type getItemResponseMessage struct {
	ResponseClass string `xml:"ResponseClass,attr"`
	ResponseCode  string `xml:"ResponseCode"`
	MessageText   string `xml:"MessageText"`
	Items         struct {
		Messages []rawMessage `xml:"Message"`
	} `xml:"Items"`
}

type rawMessage struct {
	ItemID struct {
		Id        string `xml:"Id,attr"`
		ChangeKey string `xml:"ChangeKey,attr"`
	} `xml:"ItemId"`
	Subject          string       `xml:"Subject"`
	DateTimeReceived string       `xml:"DateTimeReceived"`
	DateTimeSent     string       `xml:"DateTimeSent"`
	HasAttachments   string       `xml:"HasAttachments"`
	IsRead           string       `xml:"IsRead"`
	From             addressBlock `xml:"From"`
	Sender           addressBlock `xml:"Sender"`
	ToRecipients     addressBlock `xml:"ToRecipients"`
	CcRecipients     addressBlock `xml:"CcRecipients"`
	BccRecipients    addressBlock `xml:"BccRecipients"`
	Body             struct {
		BodyType string `xml:"BodyType,attr"`
		Inner    string `xml:",innerxml"`
	} `xml:"Body"`
	Attachments struct {
		FileAttachment []struct {
			Name        string `xml:"Name"`
			Size        uint64 `xml:"Size"`
			ContentType string `xml:"ContentType"`
		} `xml:"FileAttachment"`
	} `xml:"Attachments"`
}

type addressBlock struct {
	Mailbox []struct {
		Name         string `xml:"Name"`
		EmailAddress string `xml:"EmailAddress"`
	} `xml:"Mailbox"`
}

func parseGetItemResponse(raw []byte) ([]Detail, error) {
	var env getItemEnvelope
	if err := xml.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("parse GetItem: %w", err)
	}
	var out []Detail
	var errs []string
	for _, msg := range env.Body.GetItemResponse.ResponseMessages.Messages {
		if strings.EqualFold(strings.TrimSpace(msg.ResponseClass), "Success") {
			for _, m := range msg.Items.Messages {
				out = append(out, toDetail(m))
			}
			continue
		}
		code := strings.TrimSpace(msg.ResponseCode)
		text := strings.TrimSpace(msg.MessageText)
		if code != "" || text != "" {
			errs = append(errs, fmt.Sprintf("%s: %s", code, text))
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	return out, nil
}

func toDetail(m rawMessage) Detail {
	from := firstAddress(m.From)
	if from == "" {
		from = firstAddress(m.Sender)
	}
	recv := strings.TrimSpace(m.DateTimeReceived)
	if recv == "" {
		recv = strings.TrimSpace(m.DateTimeSent)
	}
	var atts []AttachmentInfo
	for _, fa := range m.Attachments.FileAttachment {
		n := strings.TrimSpace(fa.Name)
		if n == "" {
			continue
		}
		atts = append(atts, AttachmentInfo{
			Name:        n,
			Size:        fa.Size,
			ContentType: strings.TrimSpace(fa.ContentType),
		})
	}
	return Detail{
		ItemID:           strings.TrimSpace(m.ItemID.Id),
		ChangeKey:        strings.TrimSpace(m.ItemID.ChangeKey),
		Subject:          strings.TrimSpace(m.Subject),
		From:             from,
		To:               listAddresses(m.ToRecipients),
		Cc:               listAddresses(m.CcRecipients),
		Bcc:              listAddresses(m.BccRecipients),
		DateTimeReceived: recv,
		HasAttachments:   parseBool(m.HasAttachments),
		Attachments:      atts,
		IsRead:           parseBool(m.IsRead),
		BodyType:         strings.TrimSpace(m.Body.BodyType),
		Body:             strings.TrimSpace(decodeBody(m.Body.Inner)),
	}
}

func decodeBody(inner string) string {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return ""
	}
	var s string
	if err := xml.Unmarshal([]byte("<b>"+inner+"</b>"), &s); err == nil {
		return s
	}
	return inner
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1":
		return true
	default:
		return false
	}
}

func firstAddress(b addressBlock) string {
	addrs := listAddresses(b)
	if len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

func listAddresses(b addressBlock) []string {
	var out []string
	for _, m := range b.Mailbox {
		addr := strings.TrimSpace(m.EmailAddress)
		if addr == "" {
			addr = strings.TrimSpace(m.Name)
		}
		if addr != "" {
			out = append(out, addr)
		}
	}
	return out
}

// JoinAddresses formats addresses for display.
func JoinAddresses(addrs []string) string {
	return strings.Join(addrs, "; ")
}

// AllRecipients returns To + Cc + Bcc.
func AllRecipients(d Detail) []string {
	var all []string
	all = append(all, d.To...)
	all = append(all, d.Cc...)
	all = append(all, d.Bcc...)
	return all
}
