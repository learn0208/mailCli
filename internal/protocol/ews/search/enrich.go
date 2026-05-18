package search

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/tschuyebuhl/ews"

	"github.com/learn0208/mailcli/internal/protocol/ews/message"
)

// enrichRows loads message details via GetItem when FindItem returned only ids/subjects.
func enrichRows(c ews.Client, rows []Row, verbose bool) ([]Row, error) {
	if len(rows) == 0 {
		return rows, nil
	}

	// Batch GetItem first (one round trip when the server supports it).
	rows, _ = enrichRowsBatch(c, rows, verbose)

	// Some on-prem Exchange builds return only one item per batch ? fill the rest individually.
	for i := range rows {
		if !rowNeedsEnrich(rows[i]) {
			continue
		}
		enriched, err := enrichOne(c, rows[i])
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "GetItem %s: %v\n", shortID(rows[i].ItemID), err)
			}
			continue
		}
		rows[i] = enriched
	}
	return rows, nil
}

func rowNeedsEnrich(r Row) bool {
	if strings.TrimSpace(r.From) == "" && strings.TrimSpace(r.DateTimeReceived) == "" {
		return true
	}
	// FindItem Default often has From but not To/Cc/Bcc; without GetItem those stay empty.
	if len(r.To) == 0 && len(r.Cc) == 0 && len(r.Bcc) == 0 {
		return true
	}
	return false
}

func shortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 12 {
		return id
	}
	return id[:8] + "..."
}

func enrichRowsBatch(c ews.Client, rows []Row, verbose bool) ([]Row, error) {
	items := make([]ews.GetItemMessage, 0, len(rows))
	indexByID := make(map[string]int, len(rows))
	for i, r := range rows {
		id := strings.TrimSpace(r.ItemID)
		if id == "" {
			continue
		}
		indexByID[id] = i
		items = append(items, ews.GetItemMessage{
			ItemId: ews.ItemId{Id: id, ChangeKey: r.ChangeKey},
		})
	}
	if len(items) == 0 {
		return rows, nil
	}

	details, err := getItemDetails(c, items)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "batch GetItem: %v\n", err)
		}
		return rows, err
	}
	for _, d := range details {
		idx, ok := indexByID[strings.TrimSpace(d.ItemID.Id)]
		if !ok {
			continue
		}
		rows[idx] = mergeDetail(rows[idx], d)
	}
	return rows, nil
}

func enrichOne(c ews.Client, row Row) (Row, error) {
	id := strings.TrimSpace(row.ItemID)
	if id == "" {
		return row, fmt.Errorf("empty item id")
	}

	// Try with ChangeKey when present, then Id-only (empty ChangeKey in XML is invalid on Exchange).
	ck := strings.TrimSpace(row.ChangeKey)
	var tries []ews.ItemId
	if ck != "" {
		tries = append(tries, ews.ItemId{Id: id, ChangeKey: ck})
	}
	tries = append(tries, ews.ItemId{Id: id})
	var lastErr error
	for _, itemID := range tries {
		details, err := getItemDetails(c, []ews.GetItemMessage{{ItemId: itemID}})
		if err != nil {
			lastErr = err
			continue
		}
		if len(details) == 0 {
			lastErr = fmt.Errorf("empty response")
			continue
		}
		merged := mergeDetail(row, details[0])
		if !rowNeedsEnrich(merged) {
			return merged, nil
		}
		lastErr = fmt.Errorf("response missing fields")
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no response")
	}
	return row, lastErr
}

func getItemDetails(c ews.Client, items []ews.GetItemMessage) ([]messageDetail, error) {
	body, err := message.MarshalGetItemRequest(items, ews.BaseShapeDefault)
	if err != nil {
		return nil, err
	}
	raw, err := c.SendAndReceive(body)
	if err != nil {
		return nil, classifyErr(err)
	}
	return parseGetItemMessages(raw)
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
		Messages []messageDetail `xml:"Message"`
	} `xml:"Items"`
}

type messageDetail struct {
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
}

type addressBlock struct {
	Mailbox []struct {
		Name         string `xml:"Name"`
		EmailAddress string `xml:"EmailAddress"`
	} `xml:"Mailbox"`
}

func parseGetItemMessages(raw []byte) ([]messageDetail, error) {
	var env getItemEnvelope
	if err := xml.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("parse GetItem response: %w", err)
	}
	var out []messageDetail
	var errs []string
	for _, msg := range env.Body.GetItemResponse.ResponseMessages.Messages {
		if strings.EqualFold(strings.TrimSpace(msg.ResponseClass), "Success") {
			out = append(out, msg.Items.Messages...)
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

func mergeDetail(row Row, d messageDetail) Row {
	if s := strings.TrimSpace(d.Subject); s != "" {
		row.Subject = s
	}
	from := addressFrom(d.From)
	if from == "" {
		from = addressFrom(d.Sender)
	}
	if from != "" {
		row.From = from
	}
	if t := strings.TrimSpace(d.DateTimeReceived); t != "" {
		row.DateTimeReceived = t
	} else if t := strings.TrimSpace(d.DateTimeSent); t != "" {
		row.DateTimeReceived = t
	}
	if d.HasAttachments != "" {
		row.HasAttachments = parseBool(d.HasAttachments)
		row.hasAttKnown = true
	}
	if d.IsRead != "" {
		row.IsRead = parseBool(d.IsRead)
		row.isReadKnown = true
	}
	if ck := strings.TrimSpace(d.ItemID.ChangeKey); ck != "" {
		row.ChangeKey = ck
	}
	row.To = listAddresses(d.ToRecipients)
	row.Cc = listAddresses(d.CcRecipients)
	row.Bcc = listAddresses(d.BccRecipients)
	return row
}

func addressFrom(f addressBlock) string {
	if len(f.Mailbox) == 0 {
		return ""
	}
	if addr := strings.TrimSpace(f.Mailbox[0].EmailAddress); addr != "" {
		return addr
	}
	return strings.TrimSpace(f.Mailbox[0].Name)
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
