package search

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tschuyebuhl/ews"
)

// SentCopyHit is the outcome of looking for a just-sent message copy in Sent Items.
type SentCopyHit struct {
	Found bool
	Row   Row
	Hint  string // friendly text when Found is false (not an error)
}

// VerifySentCopy searches Sent Items for a message matching subject and optional recipients.
// Uses subject-only FindItem (reliable on on-prem) then filters by time in-process.
func VerifySentCopy(c ews.Client, ref time.Time, wantSubject string, matchTo []string, window time.Duration, verbose bool) (SentCopyHit, error) {
	wantSubject = strings.TrimSpace(wantSubject)
	if wantSubject == "" {
		return SentCopyHit{Hint: "未提供主题，跳过已发送复核。"}, nil
	}
	wantLower := strings.ToLower(wantSubject)

	// Subject-only FindItem in sentitems — avoid DateTimeReceived/To restrictions that often return 0 rows.
	opts := Options{
		Folder:              "sentitems",
		Subject:             wantSubject,
		NoDefaultDateWindow: true,
		Limit:               40,
		Verbose:             verbose,
	}
	rows, err := FindItems(c, opts)
	if err != nil {
		return SentCopyHit{}, err
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "sent verify: subject search returned %d row(s)\n", len(rows))
	}
	if hit := pickSentCopy(rows, wantLower, matchTo, ref, window); hit.Found {
		return hit, nil
	}

	// Fallback: list newest sent mail (no subject filter) and match in-process.
	opts = Options{
		Folder:              "sentitems",
		NoDefaultDateWindow: true,
		Limit:               25,
		Verbose:             verbose,
	}
	rows, err = FindItems(c, opts)
	if err != nil {
		return SentCopyHit{}, err
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "sent verify: recent sent fallback returned %d row(s)\n", len(rows))
	}
	if hit := pickSentCopy(rows, wantLower, matchTo, ref, window); hit.Found {
		return hit, nil
	}

	return SentCopyHit{
		Hint: "未在「已发送」中找到与本次主题一致的邮件。若网页邮箱里已有副本，可能是 EWS 索引延迟；请稍后在「已发送」中确认。",
	}, nil
}

func pickSentCopy(rows []Row, wantSubjectLower string, matchTo []string, ref time.Time, window time.Duration) SentCopyHit {
	for _, r := range rows {
		if strings.ToLower(strings.TrimSpace(r.Subject)) != wantSubjectLower {
			continue
		}
		if !rowWithinWindow(r, ref, window) {
			continue
		}
		if !rowMatchesRecipients(r, matchTo) {
			continue
		}
		return SentCopyHit{Found: true, Row: r}
	}
	return SentCopyHit{}
}

func rowWithinWindow(r Row, ref time.Time, window time.Duration) bool {
	ts := strings.TrimSpace(r.DateTimeReceived)
	if ts == "" {
		// FindItem IdOnly may omit time; accept subject match in recent batch.
		return true
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return true
	}
	start := ref.Add(-window)
	end := ref.Add(3 * time.Minute)
	return !t.Before(start) && !t.After(end)
}

func rowMatchesRecipients(r Row, want []string) bool {
	if len(want) == 0 {
		return true
	}
	// FindItem often does not return To; GetItem enrich may be skipped when DateTimeReceived is set.
	if len(r.To) == 0 {
		return true
	}
	blob := strings.ToLower(strings.Join(r.To, " "))
	for _, w := range want {
		w = strings.TrimSpace(strings.ToLower(w))
		if w == "" {
			continue
		}
		if strings.Contains(blob, w) {
			return true
		}
	}
	return false
}

// FormatReceivedShort returns a compact local time for CLI hints.
func FormatReceivedShort(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local().Format("2006-01-02 15:04")
	}
	if utf8.RuneCountInString(s) >= 16 {
		return s[:16]
	}
	return s
}
