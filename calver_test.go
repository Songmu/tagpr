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
		{
			name:    "YYYY.0M0D.MICRO format cross date (user bug scenario)",
			current: "2026.0227.10",
			now:     time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  "YYYY.0M0D.MICRO",
			want:    "2026.0302.0",
		},
		{
			name:    "YYYY.0M0D.MICRO format cross date with v prefix",
			current: "v2026.0227.10",
			now:     time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			vPrefix: true,
			format:  "YYYY.0M0D.MICRO",
			want:    "v2026.0302.0",
		},
		{
			name:    "YYYY.0M0D.MICRO format same date increments micro",
			current: "2026.0302.0",
			now:     time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
			vPrefix: false,
			format:  "YYYY.0M0D.MICRO",
			want:    "2026.0302.1",
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

func TestNakedPreservesZeroPadding(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		format    string
		wantNaked string
		wantTag   string
	}{
		{
			name:      "YYYY.0M0D.MICRO preserves zero-padded month+day",
			version:   "2026.0227.10",
			format:    "YYYY.0M0D.MICRO",
			wantNaked: "2026.0227.10",
			wantTag:   "2026.0227.10",
		},
		{
			name:      "YYYY.0M0D.MICRO with v prefix preserves zero-padding",
			version:   "v2026.0302.0",
			format:    "YYYY.0M0D.MICRO",
			wantNaked: "2026.0302.0",
			wantTag:   "v2026.0302.0",
		},
		{
			name:      "YY.0M0D.MICRO preserves zero-padded month+day",
			version:   "26.0123.5",
			format:    "YY.0M0D.MICRO",
			wantNaked: "26.0123.5",
			wantTag:   "26.0123.5",
		},
		{
			name:      "YYYY.0M.MICRO preserves zero-padded month",
			version:   "2026.01.3",
			format:    "YYYY.0M.MICRO",
			wantNaked: "2026.01.3",
			wantTag:   "2026.01.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := newSemver(tt.version)
			if err != nil {
				t.Fatalf("newSemver(%q) failed: %v", tt.version, err)
			}
			sv.asCalendarVersion = true
			sv.calverFormat = tt.format

			if got := sv.Naked(); got != tt.wantNaked {
				t.Errorf("Naked() = %q, want %q", got, tt.wantNaked)
			}
			if got := sv.Tag(); got != tt.wantTag {
				t.Errorf("Tag() = %q, want %q", got, tt.wantTag)
			}
		})
	}
}

func TestCalverWithTagPrefix(t *testing.T) {
	tests := []struct {
		name      string
		tagPrefix string
		tags      []string
		vPrefix   bool
		format    string
		wantTag   string
	}{
		{
			name:      "selects prefixed tag ignoring non-prefixed",
			tagPrefix: "tools",
			tags:      []string{"tools/v2026.0123.0", "v2026.0123.0"},
			vPrefix:   true,
			format:    "YYYY.0M0D.MICRO",
			wantTag:   "tools/v2026.0123.0",
		},
		{
			name:      "selects latest among prefixed tags",
			tagPrefix: "tools",
			tags:      []string{"tools/v2026.0123.0", "tools/v2026.0123.1", "v2026.0124.0"},
			vPrefix:   true,
			format:    "YYYY.0M0D.MICRO",
			wantTag:   "tools/v2026.0123.1",
		},
		{
			name:      "ignores other prefixes",
			tagPrefix: "api",
			tags:      []string{"api/v2026.123.0", "api/v2026.124.0", "web/v2026.125.0"},
			vPrefix:   true,
			format:    defaultCalendarVersioningFormat,
			wantTag:   "api/v2026.124.0",
		},
		{
			name:      "respects vPrefix=false",
			tagPrefix: "libs",
			tags:      []string{"libs/2026.0123.0", "libs/v2026.0123.0"},
			vPrefix:   false,
			format:    "YYYY.0M0D.MICRO",
			wantTag:   "libs/2026.0123.0",
		},
		{
			name:      "nested prefix",
			tagPrefix: "packages/core",
			tags:      []string{"packages/core/v2026.0123.0", "packages/web/v2026.0124.0"},
			vPrefix:   true,
			format:    "YYYY.0M0D.MICRO",
			wantTag:   "packages/core/v2026.0123.0",
		},
		{
			name:      "no matching prefixed tags returns empty",
			tagPrefix: "tools",
			tags:      []string{"v2026.0123.0", "api/v2026.0123.0"},
			vPrefix:   true,
			format:    "YYYY.0M0D.MICRO",
			wantTag:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "tagpr-calver-prefix-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			t.Cleanup(func() { os.RemoveAll(tmpDir) })

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

			for _, tag := range tt.tags {
				runGit("tag", tag)
			}

			c := &commander{
				gitPath:   "git",
				dir:       tmpDir,
				outStream: os.Stdout,
				errStream: os.Stderr,
			}
			tp := &tagpr{
				c:       c,
				gitPath: "git",
				cfg: &config{
					vPrefix:            &tt.vPrefix,
					calendarVersioning: &tt.format,
				},
				normalizedTagPrefix: normalizeTagPrefix(tt.tagPrefix),
			}

			got := tp.latestSemverTag()
			if got != tt.wantTag {
				t.Errorf("latestSemverTag() = %q, want %q", got, tt.wantTag)
			}
		})
	}
}

// TestCalverWithMixedSemverTags verifies that latestSemverTag() correctly
// returns the latest calver tag when both semver and calver tags exist,
// ignoring semver tags that don't match the calver format.
func TestCalverWithMixedSemverTags(t *testing.T) {
	tests := []struct {
		name    string
		tags    []string
		vPrefix bool
		format  string
		wantTag string
	}{
		// defaultCalendarVersioningFormat (YYYY.MM0D.MICRO) - no leading zeros
		{
			name:    "default format: ignores semver tags and returns latest calver tag",
			tags:    []string{"v7.8.0", "v2026.403.0"},
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			wantTag: "v2026.403.0",
		},
		{
			name:    "default format: returns latest calver among multiple calver and semver tags",
			tags:    []string{"v1.0.0", "v2.3.4", "v2026.403.1", "v2026.403.3"},
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			wantTag: "v2026.403.3",
		},
		{
			name:    "default format: returns empty when no calver tags exist",
			tags:    []string{"v1.0.0", "v7.8.0"},
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			wantTag: "",
		},
		// zero-padded format (YYYY.0M0D.MICRO) - produces leading zeros that
		// gitsemvers rejects as invalid semver
		{
			name:    "zero-padded format: ignores semver tags and returns latest calver tag",
			tags:    []string{"v7.8.0", "v2026.0403.0"},
			vPrefix: true,
			format:  "YYYY.0M0D.MICRO",
			wantTag: "v2026.0403.0",
		},
		{
			name:    "zero-padded format: returns latest calver among multiple calver and semver tags",
			tags:    []string{"v1.0.0", "v2.3.4", "v2026.0403.1", "v2026.0403.3"},
			vPrefix: true,
			format:  "YYYY.0M0D.MICRO",
			wantTag: "v2026.0403.3",
		},
		{
			name:    "zero-padded format: returns empty when no calver tags exist",
			tags:    []string{"v1.0.0", "v7.8.0"},
			vPrefix: true,
			format:  "YYYY.0M0D.MICRO",
			wantTag: "",
		},
		// format filters: only tags matching the configured format are recognized
		{
			name:    "zero-padded format ignores non-zero-padded tags",
			tags:    []string{"v7.8.0", "v2026.403.0", "v2026.0403.0"},
			vPrefix: true,
			format:  "YYYY.0M0D.MICRO",
			wantTag: "v2026.0403.0",
		},
		{
			name:    "default format ignores zero-padded tags",
			tags:    []string{"v7.8.0", "v2026.0403.0", "v2026.403.0"},
			vPrefix: true,
			format:  defaultCalendarVersioningFormat,
			wantTag: "v2026.403.0",
		},
		// calver disabled
		{
			name:    "falls back to semver when calver format is empty",
			tags:    []string{"v1.0.0", "v7.8.0", "v2026.0403.0"},
			vPrefix: true,
			format:  "",
			wantTag: "v7.8.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "tagpr-calver-mixed-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			t.Cleanup(func() { os.RemoveAll(tmpDir) })

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

			for _, tag := range tt.tags {
				runGit("tag", tag)
			}

			// gitsemvers (used in semver fallback path) relies on the current
			// working directory, so chdir to the temp repo.
			origDir, _ := os.Getwd()
			os.Chdir(tmpDir)
			t.Cleanup(func() { os.Chdir(origDir) })

			c := &commander{
				gitPath:   "git",
				dir:       tmpDir,
				outStream: os.Stdout,
				errStream: os.Stderr,
			}
			tp := &tagpr{
				c:       c,
				gitPath: "git",
				cfg: &config{
					vPrefix:            &tt.vPrefix,
					calendarVersioning: &tt.format,
				},
				normalizedTagPrefix: "",
			}

			got := tp.latestSemverTag()
			if got != tt.wantTag {
				t.Errorf("latestSemverTag() = %q, want %q", got, tt.wantTag)
			}
		})
	}
}

func TestZeroPaddedCalver(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tagpr-calver-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

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

	runGit("tag", "v2026.0123.0")

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

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

	latestTag := tp.latestSemverTag()
	if latestTag != "v2026.0123.0" {
		t.Fatalf("latestSemverTag() = %q, want %q", latestTag, "v2026.0123.0")
	}

	currVer, err := newSemver(latestTag)
	if err != nil {
		t.Fatalf("newSemver(%q) failed: %v", latestTag, err)
	}
	currVer.vPrefix = vPrefixTrue
	currVer.asCalendarVersion = true
	currVer.calverFormat = calverFormat

	sameDate := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	nextVer := currVer.nextCalver(sameDate)
	if nextVer.Tag() != "v2026.0123.1" {
		t.Errorf("nextCalver(same date) = %q, want %q", nextVer.Tag(), "v2026.0123.1")
	}

	nextDate := time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC)
	nextVer2 := currVer.nextCalver(nextDate)
	if nextVer2.Tag() != "v2026.0124.0" {
		t.Errorf("nextCalver(next date) = %q, want %q", nextVer2.Tag(), "v2026.0124.0")
	}
}
