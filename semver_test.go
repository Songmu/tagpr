package tagpr

import (
	"testing"
)

// TestNewCalendarVersionWithNonSemverPrefix reproduces a crash reported by a
// user whose CalVer format embeds a literal, non-numeric prefix (e.g.
// "release_YYYY0M0D.MICRO" -> "release_20260513.0"). Such a value is a valid
// CalVer string but is not parseable by github.com/Masterminds/semver/v3,
// so building the "current version" via newSemver() fails even though
// CalendarVersioning is enabled.
func TestNewCalendarVersionWithNonSemverPrefix(t *testing.T) {
	tests := []struct {
		name       string
		v          string
		format     string
		wantTag    string
		wantVPfx   bool
		wantParsed bool
	}{
		{
			name:       "literal prefix, not valid semver",
			v:          "release_20260513.0",
			format:     "release_YYYY0M0D.MICRO",
			wantTag:    "release_20260513.0",
			wantVPfx:   false,
			wantParsed: true,
		},
		{
			name:       "v-prefixed literal prefix",
			v:          "vrelease_20260513.0",
			format:     "release_YYYY0M0D.MICRO",
			wantTag:    "vrelease_20260513.0",
			wantVPfx:   true,
			wantParsed: true,
		},
		{
			name:       "unparseable current version falls back gracefully",
			v:          "v0.0.0",
			format:     "release_YYYY0M0D.MICRO",
			wantTag:    "v0.0.0",
			wantVPfx:   true,
			wantParsed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv := newCalendarVersion(tt.v, tt.format)
			if !sv.asCalendarVersion {
				t.Fatalf("newCalendarVersion(%q).asCalendarVersion should be true", tt.v)
			}
			if sv.vPrefix != tt.wantVPfx {
				t.Errorf("newCalendarVersion(%q).vPrefix = %v, want %v", tt.v, sv.vPrefix, tt.wantVPfx)
			}
			if got := sv.Tag(); got != tt.wantTag {
				t.Errorf("newCalendarVersion(%q).Tag() = %s, want %s", tt.v, got, tt.wantTag)
			}
			if (sv.cv != nil) != tt.wantParsed {
				t.Errorf("newCalendarVersion(%q) parsed cv = %v, want parsed = %v", tt.v, sv.cv != nil, tt.wantParsed)
			}
		})
	}
}

// TestNewCalendarVersionGuessNext ensures a currVer built from a non-semver
// CalVer string can still compute the next version via GuessNext, which is
// what tagpr.Run() relies on immediately after constructing currVer.
func TestNewCalendarVersionGuessNext(t *testing.T) {
	sv := newCalendarVersion("release_20260513.0", "release_YYYY0M0D.MICRO")
	next := sv.GuessNext(nil)
	if !next.asCalendarVersion {
		t.Fatalf("GuessNext() result should remain in CalVer mode")
	}
	if next.Tag() == "" {
		t.Errorf("GuessNext().Tag() should not be empty")
	}
}
