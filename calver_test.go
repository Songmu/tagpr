package tagpr

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCalver(t *testing.T) {
	tests := []struct {
		name    string
		now     time.Time
		vPrefix bool
		format  string
		want    string
	}{
		{
			name:    "January 23, 2026 with v prefix (default format)",
			now:     time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			want:    "v2026.123.0",
		},
		{
			name:    "January 23, 2026 without v prefix",
			now:     time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  defaultCalendarVersioningFormat,
			want:    "2026.123.0",
		},
		{
			name:    "December 31, 2025",
			now:     time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			want:    "v2025.1231.0",
		},
		{
			name:    "February 1, 2026",
			now:     time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  defaultCalendarVersioningFormat,
			want:    "2026.201.0",
		},
		{
			name:    "YYYY.0M.MICRO format",
			now:     time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  "YYYY.0M.MICRO",
			want:    "2026.01.0",
		},
		{
			name:    "YY.0M0D.MICRO format",
			now:     time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  "YY.0M0D.MICRO",
			want:    "26.0123.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv := newCalver(tt.now, tt.vPrefix, tt.format)
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
		format  string
		want    string
	}{
		{
			name:    "same date increments patch",
			current: "v2026.123.0",
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			want:    "v2026.123.1",
		},
		{
			name:    "same date increments patch multiple times",
			current: "v2026.123.5",
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			want:    "v2026.123.6",
		},
		{
			name:    "different day resets patch",
			current: "v2026.123.5",
			now:     time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			want:    "v2026.124.0",
		},
		{
			name:    "different month resets patch",
			current: "v2026.123.3",
			now:     time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			want:    "v2026.201.0",
		},
		{
			name:    "different year resets patch",
			current: "v2025.1231.9",
			now:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			want:    "v2026.101.0",
		},
		{
			name:    "without v prefix",
			current: "2026.123.0",
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  defaultCalendarVersioningFormat,
			want:    "2026.123.1",
		},
		{
			name:    "YYYY.0M.MICRO format same month",
			current: "2026.01.0",
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  "YYYY.0M.MICRO",
			want:    "2026.01.1",
		},
		{
			name:    "YYYY.0M.MICRO format different month",
			current: "2026.01.5",
			now:     time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  "YYYY.0M.MICRO",
			want:    "2026.02.0",
		},
		{
			name:    "YY.0M0D.MICRO format same date",
			current: "26.0123.0",
			now:     time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  "YY.0M0D.MICRO",
			want:    "26.0123.1",
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
			sv.calverFormat = tt.format

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

// TestLatestSemverTagWithZeroPaddedCalver tests that latestSemverTag()
// correctly returns zero-padded calver tags when CalVer mode is enabled.
//
// Issue: gitsemvers normalizes v2026.0123.0 -> v2026.123.0 (loses zero padding)
// because semver treats leading zeros as invalid. This breaks calver parsing
// when the format requires zero-padding (e.g., YYYY.0M0D.MICRO).
func TestLatestSemverTagWithZeroPaddedCalver(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagpr-calver-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runGit("add", "test.txt")
	runGit("commit", "-m", "initial commit")

	// Create a zero-padded calver tag (0M format for January = 01)
	runGit("tag", "v2026.0123.0")

	// gitsemvers requires being in the git directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	c := &commander{
		gitPath:   "git",
		dir:       tmpDir,
		outStream: os.Stdout,
		errStream: os.Stderr,
	}
	calverFormat := "YYYY.0M0D.MICRO"
	vPrefixTrue := true
	tp := &tagpr{
		c:       c,
		gitPath: "git",
		cfg: &config{
			vPrefix:            &vPrefixTrue,
			calendarVersioning: &calverFormat,
		},
		normalizedTagPrefix: "",
	}

	got := tp.latestSemverTag()
	expected := "v2026.0123.0"

	if got != expected {
		t.Errorf("latestSemverTag() = %q, want %q", got, expected)
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
			sv.calverFormat = defaultCalendarVersioningFormat

			// GuessNext uses time.Now() internally, so we test nextCalver directly
			next := sv.nextCalver(tt.now)
			if got := next.Tag(); got != tt.want {
				t.Errorf("nextCalver().Tag() = %s, want %s", got, tt.want)
			}
		})
	}
}
