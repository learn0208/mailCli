package search

import (
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

func TestBuildCriteriaUnread(t *testing.T) {
	c, err := buildCriteria(Options{Unread: true}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(c.WithoutFlags) != 1 || c.WithoutFlags[0] != imap.SeenFlag {
		t.Fatalf("flags: %v", c.WithoutFlags)
	}
}

func TestBuildCriteriaConflict(t *testing.T) {
	_, err := buildCriteria(Options{Unread: true, Read: true}, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildCriteriaUsesInternalDate(t *testing.T) {
	c, err := buildCriteria(Options{Since: "2026-05-01"}, time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if c.Since.IsZero() {
		t.Fatal("expected Since (internal date)")
	}
	if !c.SentSince.IsZero() {
		t.Fatal("should not use SentSince for CLI date window")
	}
}

func TestBuildCriteriaSubjectASCIIOnly(t *testing.T) {
	c, err := buildCriteria(Options{Subject: "test"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if c.Header.Get("Subject") != "test" {
		t.Fatalf("ascii subject header: %v", c.Header)
	}
	c, err = buildCriteria(Options{Subject: "招商银行"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if c.Header.Get("Subject") != "" {
		t.Fatalf("non-ascii subject must not use server SEARCH, got %v", c.Header)
	}
}

func TestBuildCriteriaSkipsDateWithTextFilter(t *testing.T) {
	c, err := buildCriteria(Options{Query: "招商银行", Since: "2026-04-01", Until: "2026-05-18"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !c.Since.IsZero() || !c.Before.IsZero() {
		t.Fatalf("text filter should skip server date unless user set flags: since=%v before=%v", c.Since, c.Before)
	}
}

func TestHasClientOnlyTextFilters(t *testing.T) {
	if !hasClientOnlyTextFilters(Options{Subject: "招商银行"}) {
		t.Fatal("expected client-only for CJK")
	}
	if hasClientOnlyTextFilters(Options{Subject: "test"}) {
		t.Fatal("ascii should allow server filter")
	}
}

func TestApplyDefaultDateWindow(t *testing.T) {
	opts := Options{DefaultDays: 3}
	ref := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	applyDefaultDateWindow(&opts, ref)
	if opts.Since == "" || opts.Until == "" {
		t.Fatal("expected default window")
	}
}
