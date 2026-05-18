package search

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"

	imapproto "github.com/learn0208/mailcli/internal/protocol/imap"
	"github.com/learn0208/mailcli/internal/timeparse"
)

// maxIMAPUIDsPerSearch caps FETCH size on large mailboxes (QQ inbox can be thousands).
const maxIMAPUIDsPerSearch = 800

// Options controls IMAP search and output.
type Options struct {
	Subject             string
	Query               string // matches subject, from, or body (any)
	Body                string
	From                string
	To                  string
	Since               string
	Until               string
	Folder              string
	Unread              bool
	Read                bool
	Attachment          bool
	Limit               int
	Output              string
	UserSetSince        bool
	UserSetUntil        bool
	NoDefaultDateWindow bool
	DefaultDays         int
	Verbose             bool
}

// Row is one message summary (item_id is UID within the selected folder).
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

// Run searches the folder and prints results.
func Run(c *imapproto.Client, opts Options) error {
	rows, err := Find(c, opts)
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

// Find executes IMAP SEARCH + FETCH and returns matching rows.
func Find(c *imapproto.Client, opts Options) ([]Row, error) {
	normalizeOptions(&opts)
	ref := time.Now()
	applyDefaultDateWindow(&opts, ref)

	mbox := imapproto.ResolveFolder(opts.Folder)
	if _, err := c.Select(mbox, false); err != nil {
		return nil, fmt.Errorf("select %q: %w", mbox, err)
	}

	criteria, err := buildCriteria(opts, ref)
	if err != nil {
		return nil, err
	}
	// Must use UidSearch: Search() returns sequence numbers, but we UidFetch by UID.
	uids, err := c.UidSearch(criteria)
	if err != nil && hasClientOnlyTextFilters(opts) {
		// QQ/163 may fail UTF-8 SEARCH; retry with date/flags only.
		fallback, ferr := buildCriteria(stripTextFilters(opts), ref)
		if ferr != nil {
			return nil, fmt.Errorf("imap search: %w", err)
		}
		uids, err = c.UidSearch(fallback)
	}
	if err != nil {
		return nil, fmt.Errorf("imap uid search: %w", err)
	}
	if len(uids) == 0 {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "imap: SEARCH returned 0 UIDs (since=%q until=%q folder=%q)\n", opts.Since, opts.Until, opts.Folder)
		}
		printSearchHint(os.Stderr, opts, 0, 0)
		return nil, nil
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "imap: SEARCH returned %d UIDs", len(uids))
		if criteria.Since.IsZero() && criteria.Before.IsZero() {
			fmt.Fprint(os.Stderr, " (no server date filter; scanning recent mail)\n")
		} else {
			fmt.Fprintf(os.Stderr, " (since=%v before=%v)\n", criteria.Since, criteria.Before)
		}
	}
	uids = trimUIDsNewestFirst(uids, maxIMAPUIDsPerSearch)

	section := &imap.BodySectionName{Peek: true}
	envelopeItems := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchUid,
		imap.FetchInternalDate,
		imap.FetchBodyStructure,
	}
	msgs, err := uidFetchMessages(c, uids, envelopeItems)
	if err != nil {
		return nil, err
	}
	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "imap: fetched %d/%d envelopes\n", len(msgs), len(uids))
	}

	bodyByUID := map[uint32]string{}
	if needBodyFetch(opts) && len(msgs) > 0 {
		bodyItems := []imap.FetchItem{imap.FetchUid, section.FetchItem()}
		bodyMsgs, err := uidFetchMessages(c, uids, bodyItems)
		if err != nil && opts.Verbose {
			fmt.Fprintf(os.Stderr, "imap: body fetch failed (%v), matching envelope only\n", err)
		} else {
			for _, bm := range bodyMsgs {
				if bm == nil {
					continue
				}
				bodyByUID[bm.Uid] = extractBodyText(bm, section)
			}
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "imap: fetched bodies for %d messages\n", len(bodyByUID))
			}
		}
	}

	var rows []Row
	var scanned []Row
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		row := messageToRow(msg)
		scanned = append(scanned, row)
		if !matchClientFilters(row, msg, opts, section, bodyByUID[msg.Uid]) {
			continue
		}
		rows = append(rows, row)
	}

	sortRowsByReceivedDesc(rows)
	if opts.Limit > 0 && len(rows) > opts.Limit {
		rows = rows[:opts.Limit]
	}
	if len(rows) == 0 {
		printSearchHint(os.Stderr, opts, len(uids), len(scanned))
		if opts.Verbose && len(scanned) > 0 {
			printScanSamples(os.Stderr, scanned, opts)
		}
	}
	return rows, nil
}

func uidFetchMessages(c *imapproto.Client, uids []uint32, items []imap.FetchItem) ([]*imap.Message, error) {
	if len(uids) == 0 {
		return nil, nil
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	ch := make(chan *imap.Message, len(uids))
	if err := c.UidFetch(seqset, items, ch); err != nil {
		return nil, fmt.Errorf("imap uid fetch: %w", err)
	}
	var out []*imap.Message
	for msg := range ch {
		if msg != nil {
			out = append(out, msg)
		}
	}
	return out, nil
}

func needBodyFetch(opts Options) bool {
	// Query often must search body: QQ IMAP envelope may omit Chinese display names.
	return strings.TrimSpace(opts.Body) != "" || strings.TrimSpace(opts.Query) != ""
}

func trimUIDsNewestFirst(uids []uint32, max int) []uint32 {
	if len(uids) <= max {
		return uids
	}
	sort.Slice(uids, func(i, j int) bool { return uids[i] > uids[j] })
	return uids[:max]
}

func stripTextFilters(opts Options) Options {
	o := opts
	o.Subject, o.From, o.Query, o.Body = "", "", "", ""
	return o
}

func hasTextFilter(opts Options) bool {
	return strings.TrimSpace(opts.Subject) != "" ||
		strings.TrimSpace(opts.From) != "" ||
		strings.TrimSpace(opts.Query) != "" ||
		strings.TrimSpace(opts.Body) != ""
}

func hasClientOnlyTextFilters(opts Options) bool {
	for _, s := range []string{opts.Subject, opts.From, opts.Query, opts.Body} {
		if strings.TrimSpace(s) != "" && !isASCIIOnly(s) {
			return true
		}
	}
	return false
}

func printSearchHint(w io.Writer, opts Options, uidCount, scanned int) {
	if w == nil {
		return
	}
	kw := firstNonEmpty(opts.Query, opts.Subject, opts.From, "关键词")
	if uidCount > 0 && scanned == 0 {
		fmt.Fprintf(w, "提示: SEARCH 找到 %d 封，但未能读取邮件内容（可重试或联系管理员）。\n", uidCount)
		return
	}
	if uidCount > 0 && scanned > 0 {
		fmt.Fprintf(w, "提示: 在 %d 封邮件中未找到匹配 %q 的结果（IMAP 信封可能无中文发件人名，已搜索正文）。\n", scanned, kw)
		fmt.Fprintf(w, "  可试: mailcli search --subject %q  或  --verbose 查看样本\n", kw)
		return
	}
	if !opts.NoDefaultDateWindow && !opts.UserSetSince && !opts.UserSetUntil {
		days := opts.DefaultDays
		if days <= 0 {
			days = 30
		}
		fmt.Fprintf(w, "提示: 默认只搜索最近 %d 天，该时间窗内没有邮件。扩大范围:\n", days)
		fmt.Fprintf(w, "  mailcli search --query %q --no-default-date-window\n", kw)
	}
	if strings.TrimSpace(opts.Subject) != "" && strings.TrimSpace(opts.From) == "" && strings.TrimSpace(opts.Query) == "" {
		fmt.Fprintf(w, "提示: 未找到主题包含 %q 的邮件。发件人关键词请用 --from 或 --query。\n", opts.Subject)
	}
}

func printScanSamples(w io.Writer, scanned []Row, opts Options) {
	fmt.Fprintf(w, "imap: sample envelope fields from scanned messages:\n")
	limit := len(scanned)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		r := scanned[i]
		fmt.Fprintf(w, "  uid=%s from=%q subject=%q\n", r.ItemID, truncate(r.From, 50), truncate(r.Subject, 40))
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	if len(vals) > 0 {
		return vals[len(vals)-1]
	}
	return ""
}

func normalizeOptions(opts *Options) {
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

func buildCriteria(opts Options, ref time.Time) (*imap.SearchCriteria, error) {
	c := imap.NewSearchCriteria()
	if opts.Unread && opts.Read {
		return nil, fmt.Errorf("cannot use both --read and --unread")
	}
	if opts.Unread {
		c.WithoutFlags = []string{imap.SeenFlag}
	}
	if opts.Read {
		c.WithFlags = []string{imap.SeenFlag}
	}
	applyDate := !hasTextFilter(opts) || opts.UserSetSince || opts.UserSetUntil || opts.NoDefaultDateWindow
	if applyDate {
		if s := strings.TrimSpace(opts.Since); s != "" {
			t, err := timeparse.Parse(s, ref)
			if err != nil {
				return nil, fmt.Errorf("since: %w", err)
			}
			c.Since = t
		}
		if s := strings.TrimSpace(opts.Until); s != "" {
			t, err := timeparse.Parse(s, ref)
			if err != nil {
				return nil, fmt.Errorf("until: %w", err)
			}
			c.Before = t.Add(24 * time.Hour)
		}
	}
	// Server-side prefilter only for ASCII (QQ/163 reject UTF-8 SEARCH literals).
	if subj := strings.TrimSpace(opts.Subject); subj != "" && isASCIIOnly(subj) {
		c.Header.Add("Subject", subj)
	}
	if from := strings.TrimSpace(opts.From); from != "" && isASCIIOnly(from) {
		c.Header.Add("From", from)
	}
	if q := strings.TrimSpace(opts.Query); q != "" && isASCIIOnly(q) {
		c.Text = []string{q}
	}
	return c, nil
}

func messageToRow(msg *imap.Message) Row {
	row := Row{
		ItemID: fmt.Sprintf("%d", msg.Uid),
	}
	if msg.Envelope != nil {
		row.Subject = decodeDisplayName(msg.Envelope.Subject)
		if msg.Envelope.From != nil && len(msg.Envelope.From) > 0 {
			row.From = formatAddress(msg.Envelope.From[0])
		}
		row.To = formatAddresses(msg.Envelope.To)
		row.Cc = formatAddresses(msg.Envelope.Cc)
		row.Bcc = formatAddresses(msg.Envelope.Bcc)
		if !msg.InternalDate.IsZero() {
			row.DateTimeReceived = msg.InternalDate.UTC().Format(time.RFC3339)
		} else if !msg.Envelope.Date.IsZero() {
			row.DateTimeReceived = msg.Envelope.Date.UTC().Format(time.RFC3339)
		}
	}
	if msg.Flags != nil {
		row.isReadKnown = true
		row.IsRead = containsFlag(msg.Flags, imap.SeenFlag)
	}
	if msg.BodyStructure != nil {
		row.hasAttKnown = true
		row.HasAttachments = hasAttachments(msg.BodyStructure)
	}
	return row
}

func matchClientFilters(row Row, msg *imap.Message, opts Options, section *imap.BodySectionName, bodyText string) bool {
	if q := strings.TrimSpace(opts.Query); q != "" {
		if !textMatches(row, bodyText, q) {
			return false
		}
	}
	if subj := strings.TrimSpace(opts.Subject); subj != "" {
		if !strings.Contains(strings.ToLower(row.Subject), strings.ToLower(subj)) {
			return false
		}
	}
	if from := strings.TrimSpace(opts.From); from != "" {
		if !strings.Contains(strings.ToLower(row.From), strings.ToLower(from)) {
			return false
		}
	}
	if to := strings.TrimSpace(opts.To); to != "" {
		if !recipientContains(row.To, row.Cc, row.Bcc, to) {
			return false
		}
	}
	if opts.Attachment && row.hasAttKnown && !row.HasAttachments {
		return false
	}
	if bodyQ := strings.TrimSpace(opts.Body); bodyQ != "" {
		if !textMatches(row, bodyText, bodyQ) {
			return false
		}
	}
	return true
}

func textMatches(row Row, bodyText, needle string) bool {
	needle = strings.ToLower(strings.TrimSpace(needle))
	hay := strings.ToLower(row.Subject + " " + row.From + " " + bodyText)
	return strings.Contains(hay, needle)
}

func extractBodyText(msg *imap.Message, section *imap.BodySectionName) string {
	if msg == nil {
		return ""
	}
	r := msg.GetBody(section)
	if r == nil {
		return ""
	}
	ent, err := mail.CreateReader(r)
	if err != nil {
		return ""
	}
	var parts []string
	for {
		p, err := ent.NextPart()
		if err != nil {
			break
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			b, _ := readAllLimited(p.Body, 256*1024)
			parts = append(parts, string(b))
		case *mail.AttachmentHeader:
			continue
		default:
			_ = h
		}
	}
	return strings.Join(parts, "\n")
}

func readAllLimited(r interface{ Read([]byte) (int, error) }, max int) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for len(buf) < max {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	if len(buf) > max {
		buf = buf[:max]
	}
	return buf, nil
}

func hasAttachments(bs *imap.BodyStructure) bool {
	if bs == nil {
		return false
	}
	if len(bs.Parts) == 0 {
		return false
	}
	if strings.EqualFold(bs.MIMEType, "multipart") && strings.EqualFold(bs.MIMESubType, "mixed") {
		return len(bs.Parts) > 0
	}
	for _, p := range bs.Parts {
		if p == nil {
			continue
		}
		if strings.EqualFold(p.MIMEType, "multipart") {
			if hasAttachments(p) {
				return true
			}
			continue
		}
		if !strings.EqualFold(p.MIMEType, "text") {
			return true
		}
	}
	return false
}

func containsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, flag) {
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
		return fmt.Sprintf("%s <%s@%s>", decodeDisplayName(addr.PersonalName), addr.MailboxName, addr.HostName)
	}
	if addr.PersonalName != "" {
		return decodeDisplayName(addr.PersonalName)
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

func recipientContains(to, cc, bcc []string, needle string) bool {
	needle = strings.ToLower(needle)
	for _, list := range [][]string{to, cc, bcc} {
		for _, a := range list {
			if strings.Contains(strings.ToLower(a), needle) {
				return true
			}
		}
	}
	return false
}

func sortRowsByReceivedDesc(rows []Row) {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if parseReceived(rows[j].DateTimeReceived).After(parseReceived(rows[i].DateTimeReceived)) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

func parseReceived(s string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}
	}
	return t
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
		return "Y"
	}
	return "N"
}
