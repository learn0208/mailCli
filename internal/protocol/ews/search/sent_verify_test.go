package search

import (
	"testing"
	"time"
)

func TestRowMatchesRecipients_emptyToPasses(t *testing.T) {
	r := Row{Subject: "x", To: nil}
	if !rowMatchesRecipients(r, []string{"a@b.c"}) {
		t.Fatal("empty To should not reject verify match")
	}
}

func TestRowMatchesRecipients_withTo(t *testing.T) {
	r := Row{To: []string{"36291161@qq.com"}}
	if !rowMatchesRecipients(r, []string{"36291161@qq.com"}) {
		t.Fatal("expected match")
	}
	if rowMatchesRecipients(r, []string{"other@x.com"}) {
		t.Fatal("expected no match")
	}
}

func TestRowWithinWindow(t *testing.T) {
	ref := time.Date(2026, 5, 15, 16, 30, 0, 0, time.UTC)
	r := Row{DateTimeReceived: ref.Add(-2 * time.Minute).Format(time.RFC3339)}
	if !rowWithinWindow(r, ref, 30*time.Minute) {
		t.Fatal("should be in window")
	}
	r.DateTimeReceived = ref.Add(-2 * time.Hour).Format(time.RFC3339)
	if rowWithinWindow(r, ref, 30*time.Minute) {
		t.Fatal("should be outside window")
	}
}

func TestPickSentCopy_subjectAndTime(t *testing.T) {
	ref := time.Now()
	subj := "ewsCli send email"
	rows := []Row{{
		Subject:          subj,
		ItemID:           "id1",
		DateTimeReceived: ref.Add(-1 * time.Minute).Format(time.RFC3339),
	}}
	hit := pickSentCopy(rows, "ewscli send email", nil, ref, 30*time.Minute)
	if !hit.Found || hit.Row.ItemID != "id1" {
		t.Fatalf("expected hit, got %+v", hit)
	}
}
