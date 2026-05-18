package timeparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var agoRE = regexp.MustCompile(`(?i)^\s*(\d+)\s*(day|days|week|weeks|hour|hours)\s+ago\s*$`)

// Parse parses RFC3339, date-only, or relative strings like "7 days ago" against ref (usually time.Now()).
func Parse(s string, ref time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}
	if m := agoRE.FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil || n <= 0 {
			return time.Time{}, fmt.Errorf("invalid relative amount in %q", s)
		}
		unit := strings.ToLower(m[2])
		switch {
		case strings.HasPrefix(unit, "day"):
			return ref.AddDate(0, 0, -n), nil
		case strings.HasPrefix(unit, "week"):
			return ref.AddDate(0, 0, -7*n), nil
		case strings.HasPrefix(unit, "hour"):
			return ref.Add(-time.Duration(n) * time.Hour), nil
		}
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.DateOnly,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s)
}
