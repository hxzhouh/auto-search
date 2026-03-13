package extractor

import (
	"testing"
	"time"
)

func TestParsePublishedAt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input    string
		wantNil  bool
		wantTime time.Time
	}{
		{"2026-03-10T12:00:00Z", false, time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)},
		{"2026-03-10", false, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)},
		{"not-a-date", true, time.Time{}},
		{"", true, time.Time{}},
	}

	for _, tc := range cases {
		got := parsePublishedAt(tc.input)
		if tc.wantNil {
			if got != nil {
				t.Errorf("parsePublishedAt(%q) = %v, want nil", tc.input, got)
			}
		} else {
			if got == nil {
				t.Errorf("parsePublishedAt(%q) = nil, want %v", tc.input, tc.wantTime)
			} else if !got.Equal(tc.wantTime) {
				t.Errorf("parsePublishedAt(%q) = %v, want %v", tc.input, got, tc.wantTime)
			}
		}
	}
}
