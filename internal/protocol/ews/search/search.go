package search

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/tschuyebuhl/ews"

	"github.com/learn0208/mailcli/internal/timeparse"
	"github.com/learn0208/mailcli/internal/protocol/ews/xmlutil"
)

// Options controls FindItem query and output.
type Options struct {
	Subject    string
	Body       string
	From       string
	To         string
	Since      string
	Until      string
	Folder     string
	Unread     bool
	Read       bool
	Attachment bool
	Limit      int
	Output     string
	Verbose    bool
	// UserSetSince / UserSetUntil are true when the user passed the corresponding flag on the CLI
	// (even if empty), so we do not silently apply a default date window.
	UserSetSince bool
	UserSetUntil bool
	// NoDefaultDateWindow skips applying the default received-time window when since/until are omitted.
	NoDefaultDateWindow bool
	// DefaultDays is the length of that window in calendar days (default 7). Used only when the default window applies.
	DefaultDays int
}

func normalizeSearchOptions(opts *Options) {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.DefaultDays <= 0 {
		opts.DefaultDays = 7
	}
}

func applyDefaultDateWindow(opts *Options, ref time.Time) {
	if opts.NoDefaultDateWindow || opts.UserSetSince || opts.UserSetUntil {
		return
	}
	if strings.TrimSpace(opts.Since) != "" || strings.TrimSpace(opts.Until) != "" {
		return
	}
	days := opts.DefaultDays
	if days <= 0 {
		days = 7
	}
	opts.Since = ref.AddDate(0, 0, -days).Format(time.RFC3339)
	opts.Until = ref.Format(time.RFC3339)
}

// sortOrderDescendingReceivedXML orders FindItem results by DateTimeReceived descending (newest first).
const sortOrderDescendingReceivedXML = `<m:SortOrder>
  <t:FieldOrder Order="Descending">
    <t:FieldURI FieldURI="item:DateTimeReceived"/>
  </t:FieldOrder>
</m:SortOrder>`

func insertSortOrderAfterItemShape(xmlBytes []byte) []byte {
	s := string(xmlBytes)
	needle := "</m:ItemShape>"
	i := strings.Index(s, needle)
	if i < 0 {
		return xmlBytes
	}
	i += len(needle)
	return []byte(s[:i] + "\n" + sortOrderDescendingReceivedXML + s[i:])
}

// Row is one message summary for JSON/table output.
type Row struct {
	ItemID           string   `json:"item_id"`
	ChangeKey        string   `json:"change_key,omitempty"`
	Subject          string   `json:"subject"`
	From             string   `json:"from"`
	To               []string `json:"to"`
	Cc               []string `json:"cc,omitempty"`
	Bcc              []string `json:"bcc,omitempty"`
	DateTimeReceived string   `json:"datetime_received"`
	HasAttachments   bool     `json:"has_attachments"`
	IsRead           bool     `json:"is_read"`
	hasAttKnown      bool     `json:"-"`
	isReadKnown      bool     `json:"-"`
}

// Run executes FindItem and prints results.
func Run(c ews.Client, opts Options) error {
	rows, err := FindItems(c, opts)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(opts.Output)) {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ITEM_ID\tSUBJECT\tFROM\tTO\tRECEIVED\tATT\tREAD")
		for _, r := range rows {
			to := formatRecipientColumn(r)
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				r.ItemID,
				truncate(r.Subject, 36),
				truncate(r.From, 26),
				truncate(to, 40),
				formatReceived(r.DateTimeReceived),
				formatBool(r.HasAttachments, r.hasAttKnown),
				formatBool(r.IsRead, r.isReadKnown),
			)
		}
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "%d message(s)\n", len(rows))
		return nil
	}
}

func formatRecipientColumn(r Row) string {
	var parts []string
	if len(r.To) > 0 {
		parts = append(parts, "To:"+strings.Join(r.To, ","))
	}
	if len(r.Cc) > 0 {
		parts = append(parts, "Cc:"+strings.Join(r.Cc, ","))
	}
	if len(r.Bcc) > 0 {
		parts = append(parts, "Bcc:"+strings.Join(r.Bcc, ","))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}

func truncate(s string, maxRunes int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes-1]) + "..."
}

func formatReceived(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local().Format("2006-01-02 15:04")
	}
	if len(s) >= 16 {
		return s[:16]
	}
	return s
}

func formatBool(v bool, known bool) string {
	if !known {
		return "-"
	}
	if v {
		return "yes"
	}
	return "no"
}

func classifyErr(err error) error {
	if err == nil {
		return nil
	}
	var se *ews.SoapError
	if errors.As(err, &se) {
		code := ""
		if se.Fault != nil {
			code = strings.TrimSpace(se.Fault.Detail.ResponseCode)
		}
		if code != "" {
			return fmt.Errorf("ews error [%s]: %s", code, strings.TrimSpace(se.Error()))
		}
		return fmt.Errorf("ews soap: %s", strings.TrimSpace(se.Error()))
	}
	var he *ews.HTTPError
	if errors.As(err, &he) {
		if he.StatusCode == 401 {
			return fmt.Errorf("authentication failed (HTTP %d)", he.StatusCode)
		}
		return fmt.Errorf("http error: %s", he.Error())
	}
	return fmt.Errorf("request failed: %w", err)
}

// ClassifyErr normalizes EWS/HTTP errors for subcommands (e.g. show).
func ClassifyErr(err error) error {
	return classifyErr(err)
}

func isInvalidRequest(err error) bool {
	var se *ews.SoapError
	if !errors.As(err, &se) || se.Fault == nil {
		return false
	}
	code := strings.TrimSpace(se.Fault.Detail.ResponseCode)
	return strings.EqualFold(code, "ErrorInvalidRequest")
}

func distinguishedFolder(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return "inbox"
	}
	switch n {
	case "inbox":
		return "inbox"
	case "sent", "sentitems", "sent items":
		return "sentitems"
	case "drafts":
		return "drafts"
	case "deleteditems", "deleted", "trash":
		return "deleteditems"
	case "junk", "junkemail":
		return "junkemail"
	default:
		return n
	}
}

// buildFindItemBody builds FindItem XML. Subject-only uses the same shape as github.com/tschuyebuhl/ews
// (IdOnly + item:Subject only). Complex filters use hand-built XML without invalid ItemShape properties.
func buildFindItemBody(opts Options, ref time.Time) ([]byte, error) {
	if isSubjectOnly(opts) {
		return marshalSubjectFindItem(opts)
	}
	if opts.Subject != "" && !hasComplexFilters(opts) {
		// subject + simple flags: try QueryString (Exchange 2013+), often more reliable than Contains on on-prem.
		return []byte(buildQueryStringFindItem(opts)), nil
	}
	xmlStr, err := buildComplexFindItemXML(opts, ref)
	if err != nil {
		return nil, err
	}
	return []byte(xmlStr), nil
}

func isSubjectOnly(opts Options) bool {
	return opts.Subject != "" &&
		opts.Body == "" && opts.From == "" && opts.To == "" &&
		opts.Since == "" && opts.Until == "" &&
		!opts.Unread && !opts.Read && !opts.Attachment
}

func hasComplexFilters(opts Options) bool {
	return opts.Body != "" || opts.From != "" || opts.To != "" ||
		opts.Since != "" || opts.Until != "" ||
		opts.Unread || opts.Read || opts.Attachment
}

// marshalSubjectFindItem matches ewsutil.FindEmail / library tests (ParentFolderIds before Restriction).
func marshalSubjectFindItem(opts Options) ([]byte, error) {
	req := &ews.FindItemRequest{
		Traversal: "Shallow",
		ItemShape: ews.ItemShape{
			BaseShape: ews.BaseShapeIdOnly,
			AdditionalProperties: ews.AdditionalProperties{
				FieldURI: []ews.FieldURI{{FieldURI: "item:Subject"}},
			},
		},
		IndexedPageItemView: &ews.IndexedPageItemView{
			MaxEntriesReturned: opts.Limit,
			Offset:             0,
			BasePoint:          ews.BasePointBeginning,
		},
		ParentFolderIds: ews.ParentFolderIds{
			DistinguishedFolderId: ews.DistinguishedFolderId{Id: distinguishedFolder(opts.Folder)},
		},
		Restriction: &ews.Restriction{
			Contains: &ews.Contains{
				BaseFiltering: ews.BaseFiltering{
					AdditionalProperties: ews.AdditionalProperties{
						FieldURI: []ews.FieldURI{{FieldURI: "item:Subject"}},
					},
				},
				Constant: []ews.Constant{
					{Value: opts.Subject},
				},
				ContainmentMode:       "Substring",
				ContainmentComparison: "IgnoreCase",
			},
		},
	}
	body, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, err
	}
	return insertSortOrderAfterItemShape(body), nil
}

func buildQueryStringFindItem(opts Options) string {
	folderID := distinguishedFolder(opts.Folder)
	page := fmt.Sprintf(`<m:IndexedPageItemView MaxEntriesReturned="%d" Offset="0" BasePoint="Beginning"/>`, opts.Limit)
	q := xmlutil.EscapeAttr(opts.Subject)
	return fmt.Sprintf(`<m:FindItem Traversal="Shallow">
<m:ItemShape>
  <t:BaseShape>Default</t:BaseShape>
</m:ItemShape>
%s
%s
<m:ParentFolderIds>
  <t:DistinguishedFolderId Id="%s"/>
</m:ParentFolderIds>
<m:QueryString>%s</m:QueryString>
</m:FindItem>`, sortOrderDescendingReceivedXML, page, xmlutil.EscapeAttr(folderID), q)
}

func buildComplexFindItemXML(opts Options, ref time.Time) (string, error) {
	var filters []string
	if opts.Subject != "" {
		filters = append(filters, containsFilter("item:Subject", opts.Subject))
	}
	if opts.Body != "" {
		filters = append(filters, containsFilter("item:Body", opts.Body))
	}
	if opts.From != "" {
		filters = append(filters, containsFilter("message:Sender", opts.From))
	}
	if opts.To != "" {
		filters = append(filters, containsFilter("message:ToRecipients", opts.To))
	}
	if opts.Since != "" {
		t, err := timeparse.Parse(opts.Since, ref)
		if err != nil {
			return "", fmt.Errorf("--since: %w", err)
		}
		filters = append(filters, dateCompare("item:DateTimeReceived", "IsGreaterThanOrEqualTo", t.UTC().Format(time.RFC3339)))
	}
	if opts.Until != "" {
		t, err := timeparse.Parse(opts.Until, ref)
		if err != nil {
			return "", fmt.Errorf("--until: %w", err)
		}
		filters = append(filters, dateCompare("item:DateTimeReceived", "IsLessThanOrEqualTo", t.UTC().Format(time.RFC3339)))
	}
	if opts.Unread && !opts.Read {
		filters = append(filters, boolEqual("item:IsRead", false))
	}
	if opts.Read && !opts.Unread {
		filters = append(filters, boolEqual("item:IsRead", true))
	}
	if opts.Attachment {
		filters = append(filters, boolEqual("item:HasAttachments", true))
	}

	var restriction string
	switch len(filters) {
	case 0:
		restriction = ""
	case 1:
		restriction = "<m:Restriction>" + filters[0] + "</m:Restriction>"
	default:
		restriction = "<m:Restriction><t:And>" + strings.Join(filters, "") + "</t:And></m:Restriction>"
	}

	folderID := distinguishedFolder(opts.Folder)
	page := fmt.Sprintf(`<m:IndexedPageItemView MaxEntriesReturned="%d" Offset="0" BasePoint="Beginning"/>`, opts.Limit)

	// Default shape only - avoid AdditionalProperties that some on-prem servers reject in FindItem.
	return fmt.Sprintf(`<m:FindItem Traversal="Shallow">
<m:ItemShape>
  <t:BaseShape>Default</t:BaseShape>
</m:ItemShape>
%s
%s
<m:ParentFolderIds>
  <t:DistinguishedFolderId Id="%s"/>
</m:ParentFolderIds>
%s
</m:FindItem>`, sortOrderDescendingReceivedXML, page, xmlutil.EscapeAttr(folderID), restriction), nil
}

func containsFilter(fieldURI, value string) string {
	v := xmlutil.EscapeAttr(value)
	return fmt.Sprintf(`<t:Contains ContainmentMode="Substring" ContainmentComparison="IgnoreCase">
  <t:FieldURI FieldURI="%s"/>
  <t:Constant Value="%s"/>
</t:Contains>`, fieldURI, v)
}

func dateCompare(fieldURI, op, iso string) string {
	v := xmlutil.EscapeAttr(iso)
	return fmt.Sprintf(`<t:%s>
  <t:FieldURI FieldURI="%s"/>
  <t:FieldURIOrConstant>
    <t:Constant Value="%s"/>
  </t:FieldURIOrConstant>
</t:%s>`, op, fieldURI, v, op)
}

func boolEqual(fieldURI string, val bool) string {
	s := "false"
	if val {
		s = "true"
	}
	return fmt.Sprintf(`<t:IsEqualTo>
  <t:FieldURI FieldURI="%s"/>
  <t:FieldURIOrConstant>
    <t:Constant Value="%s"/>
  </t:FieldURIOrConstant>
</t:IsEqualTo>`, fieldURI, s)
}

func parseFindItemResponse(bb []byte) ([]Row, error) {
	var env findEnvelope
	if err := xml.Unmarshal(bb, &env); err != nil {
		return nil, fmt.Errorf("parse find response: %w", err)
	}
	msg := env.Body.FindItemResponse.ResponseMessages.FindItemResponseMessage
	if strings.TrimSpace(msg.ResponseClass) != "" && !strings.EqualFold(msg.ResponseClass, "Success") {
		return nil, fmt.Errorf("find failed: %s (%s)", msg.MessageText, msg.ResponseCode)
	}
	if msg.RootFolder == nil {
		return nil, nil
	}
	var rows []Row
	for _, m := range msg.RootFolder.Items.Messages {
		from := ""
		if len(m.From.Mailbox) > 0 {
			from = strings.TrimSpace(m.From.Mailbox[0].EmailAddress)
			if from == "" {
				from = strings.TrimSpace(m.From.Mailbox[0].Name)
			}
		}
		rows = append(rows, Row{
			ItemID:           strings.TrimSpace(m.ItemID.Id),
			ChangeKey:        strings.TrimSpace(m.ItemID.ChangeKey),
			Subject:          strings.TrimSpace(m.Subject),
			From:             from,
			DateTimeReceived: strings.TrimSpace(m.DateTimeReceived),
			HasAttachments:   parseBool(m.HasAttachments),
			IsRead:           parseBool(m.IsRead),
			hasAttKnown:      strings.TrimSpace(m.HasAttachments) != "",
			isReadKnown:      strings.TrimSpace(m.IsRead) != "",
		})
	}
	return rows, nil
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1":
		return true
	default:
		return false
	}
}

type findEnvelope struct {
	Body struct {
		FindItemResponse struct {
			ResponseMessages struct {
				FindItemResponseMessage findItemResponseMessage `xml:"FindItemResponseMessage"`
			} `xml:"ResponseMessages"`
		} `xml:"FindItemResponse"`
	} `xml:"Body"`
}

type findItemResponseMessage struct {
	ResponseClass string `xml:"ResponseClass,attr"`
	MessageText   string `xml:"MessageText"`
	ResponseCode  string `xml:"ResponseCode"`
	RootFolder    *struct {
		Items struct {
			Messages []findMessage `xml:"Message"`
		} `xml:"Items"`
	} `xml:"RootFolder"`
}

type findMessage struct {
	ItemID struct {
		Id        string `xml:"Id,attr"`
		ChangeKey string `xml:"ChangeKey,attr"`
	} `xml:"ItemId"`
	Subject          string    `xml:"Subject"`
	DateTimeReceived string    `xml:"DateTimeReceived"`
	HasAttachments   string    `xml:"HasAttachments"`
	IsRead           string    `xml:"IsRead"`
	From             fromBlock `xml:"From"`
}

type fromBlock struct {
	Mailbox []struct {
		Name         string `xml:"Name"`
		EmailAddress string `xml:"EmailAddress"`
	} `xml:"Mailbox"`
}
