package send

import "testing"

func TestSplitAddresses(t *testing.T) {
	got := SplitAddresses("a@b.com, c@d.com , ")
	if len(got) != 2 || got[0] != "a@b.com" || got[1] != "c@d.com" {
		t.Fatalf("got %v", got)
	}
}

func TestFormatFrom(t *testing.T) {
	s := formatFrom("u@example.com", "Test User")
	if s == "" || !contains(s, "u@example.com") {
		t.Fatalf("got %q", s)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
