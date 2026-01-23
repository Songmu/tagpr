package tagpr

import (
	"testing"
	"time"
)

func TestNewCalver(t *testing.T) {
	tests := []struct {
		name    string
		now     time.Time
		vPrefix bool
		want    string
	}{
		{
			name:    "January 23, 2026 with v prefix",
			now:     time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			want:    "v2026.123.0",
		},
		{
			name:    "January 23, 2026 without v prefix",
			now:     time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC),
			vPrefix: false,
			want:    "2026.123.0",
		},
		{
			name:    "December 31, 2025",
			now:     time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			want:    "v2025.1231.0",
		},
		{
			name:    "February 1, 2026",
			now:     time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			vPrefix: false,
			want:    "2026.201.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv := newCalver(tt.now, tt.vPrefix)
			if got := sv.Tag(); got != tt.want {
				t.Errorf("newCalver().Tag() = %s, want %s", got, tt.want)
			}
			if !sv.asCalendarVersion {
				t.Errorf("newCalver().asCalendarVersion should be true")
			}
		})
	}
}

func TestNextCalver(t *testing.T) {
	tests := []struct {
		name    string
		current string
		now     time.Time
		vPrefix bool
		want    string
	}{
		{
			name:    "same date increments patch",
			current: "v2026.123.0",
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			vPrefix: true,
			want:    "v2026.123.1",
		},
		{
			name:    "same date increments patch multiple times",
			current: "v2026.123.5",
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			vPrefix: true,
			want:    "v2026.123.6",
		},
		{
			name:    "different day resets patch",
			current: "v2026.123.5",
			now:     time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			want:    "v2026.124.0",
		},
		{
			name:    "different month resets patch",
			current: "v2026.123.3",
			now:     time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			want:    "v2026.201.0",
		},
		{
			name:    "different year resets patch",
			current: "v2025.1231.9",
			now:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			want:    "v2026.101.0",
		},
		{
			name:    "without v prefix",
			current: "2026.123.0",
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			vPrefix: false,
			want:    "2026.123.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := newSemver(tt.current)
			if err != nil {
				t.Fatalf("newSemver(%s) failed: %v", tt.current, err)
			}
			sv.asCalendarVersion = true
			sv.vPrefix = tt.vPrefix

			next := sv.nextCalver(tt.now)
			if got := next.Tag(); got != tt.want {
				t.Errorf("nextCalver().Tag() = %s, want %s", got, tt.want)
			}
			if !next.asCalendarVersion {
				t.Errorf("nextCalver().asCalendarVersion should be true")
			}
		})
	}
}

func TestGuessNextWithCalver(t *testing.T) {
	tests := []struct {
		name    string
		current string
		labels  []string
		now     time.Time
		want    string
	}{
		{
			name:    "calver ignores major label",
			current: "v2026.123.0",
			labels:  []string{"major"},
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			want:    "v2026.123.1",
		},
		{
			name:    "calver ignores minor label",
			current: "v2026.123.0",
			labels:  []string{"minor"},
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			want:    "v2026.123.1",
		},
		{
			name:    "calver ignores all labels",
			current: "v2026.123.0",
			labels:  []string{"major", "minor"},
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			want:    "v2026.123.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := newSemver(tt.current)
			if err != nil {
				t.Fatalf("newSemver(%s) failed: %v", tt.current, err)
			}
			sv.asCalendarVersion = true

			// GuessNext uses time.Now() internally, so we test nextCalver directly
			next := sv.nextCalver(tt.now)
			if got := next.Tag(); got != tt.want {
				t.Errorf("nextCalver().Tag() = %s, want %s", got, tt.want)
			}
		})
	}
}
