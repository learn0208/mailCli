package timeparse

import (
	"testing"
	"time"
)

func TestParseRelativeDays(t *testing.T) {
	ref := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	got, err := Parse("7 days ago", ref)
	if err != nil {
		t.Fatal(err)
	}
	want := ref.AddDate(0, 0, -7)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestParseDateOnly(t *testing.T) {
	ref := time.Now()
	got, err := Parse("2026-05-01", ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Year() != 2026 || got.Month() != time.May || got.Day() != 1 {
		t.Fatalf("got %v", got)
	}
}
