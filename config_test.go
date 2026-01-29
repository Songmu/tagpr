package tagpr

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Songmu/gitconfig"
)

func TestConfig(t *testing.T) {
	tmpdir := t.TempDir()
	confPath := filepath.Join(tmpdir, defaultConfigFile)
	cfg := &config{
		conf:      confPath,
		gitconfig: &gitconfig.Config{GitPath: "git", File: confPath},
	}

	if err := cfg.Reload(); err != nil {
		t.Error(err)
	}
	if e, g := "", cfg.ReleaseBranch(); e != g {
		t.Errorf("got: %s, expext: %s", g, e)
	}
	if err := cfg.SetReleaseBranch("main"); err != nil {
		t.Error(err)
	}
	if e, g := "main", cfg.ReleaseBranch(); e != g {
		t.Errorf("got: %s, expext: %s", g, e)
	}
	if err := cfg.SetVersionFile(""); err != nil {
		t.Error(err)
	}
	if e, g := "", cfg.VersionFile(); e != g {
		t.Errorf("got: %s, expext: %s", g, e)
	}
	if e, g := []string{"major"}, cfg.MajorLabels(); !reflect.DeepEqual(e, g) {
		t.Errorf("got: %s, expext: %s", g, e)
	}
	if e, g := []string{"minor"}, cfg.MinorLabels(); !reflect.DeepEqual(e, g) {
		t.Errorf("got: %s, expext: %s", g, e)
	}
	if e, g := "[tagpr]", cfg.CommitPrefix(); !reflect.DeepEqual(e, g) {
		t.Errorf("got: %s, expext: %s", g, e)
	}

	b, err := os.ReadFile(confPath)
	if err != nil {
		t.Error(err)
	}

	var out string
	for line := range strings.SplitSeq(string(b), "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			out += line + "\n"
		}
	}
	expect := `[tagpr]
	releaseBranch = main
	versionFile = -
`
	if out != expect {
		t.Errorf("got:\n%s\nexpect:\n%s", out, expect)
	}
}

func TestConfigCalendarVersioning(t *testing.T) {
	tmpdir := t.TempDir()
	confPath := filepath.Join(tmpdir, defaultConfigFile)
	cfg := &config{
		conf:      confPath,
		gitconfig: &gitconfig.Config{GitPath: "git", File: confPath},
	}

	if err := cfg.Reload(); err != nil {
		t.Error(err)
	}

	if cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be false initially")
	}
	if e, g := "", cfg.CalendarVersioningFormat(); e != g {
		t.Errorf("got: %s, expect: %s", g, e)
	}

	if err := cfg.SetCalendarVersioning("true"); err != nil {
		t.Error(err)
	}
	if !cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be true")
	}
	if e, g := defaultCalendarVersioningFormat, cfg.CalendarVersioningFormat(); e != g {
		t.Errorf("got: %s, expect: %s", g, e)
	}

	if err := cfg.Reload(); err != nil {
		t.Error(err)
	}
	if !cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be true after reload")
	}

	if err := cfg.SetCalendarVersioning("false"); err != nil {
		t.Error(err)
	}
	if cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be false")
	}
	if e, g := "", cfg.CalendarVersioningFormat(); e != g {
		t.Errorf("got: %s, expect: %s", g, e)
	}

	if err := cfg.SetCalendarVersioning(""); err != nil {
		t.Error(err)
	}
	if cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be false for empty string")
	}
	if e, g := "", cfg.CalendarVersioningFormat(); e != g {
		t.Errorf("got: %s, expect: %s", g, e)
	}
}

func TestConfigCalendarVersioningWithFormat(t *testing.T) {
	tmpdir := t.TempDir()
	confPath := filepath.Join(tmpdir, defaultConfigFile)
	cfg := &config{
		conf:      confPath,
		gitconfig: &gitconfig.Config{GitPath: "git", File: confPath},
	}

	if err := cfg.Reload(); err != nil {
		t.Error(err)
	}

	if err := cfg.SetCalendarVersioning("YYYY.0M.MICRO"); err != nil {
		t.Error(err)
	}
	if !cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be true")
	}
	if e, g := "YYYY.0M.MICRO", cfg.CalendarVersioningFormat(); e != g {
		t.Errorf("got: %s, expect: %s", g, e)
	}

	if err := cfg.Reload(); err != nil {
		t.Error(err)
	}
	if e, g := "YYYY.0M.MICRO", cfg.CalendarVersioningFormat(); e != g {
		t.Errorf("got: %s, expect: %s", g, e)
	}
}

func TestConfigCalendarVersioningFromEnv(t *testing.T) {
	tmpdir := t.TempDir()
	confPath := filepath.Join(tmpdir, defaultConfigFile)

	t.Setenv("TAGPR_CALENDAR_VERSIONING", "true")

	cfg := &config{
		conf:      confPath,
		gitconfig: &gitconfig.Config{GitPath: "git", File: confPath},
	}

	if err := cfg.Reload(); err != nil {
		t.Error(err)
	}

	if !cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be true from env")
	}
	if e, g := defaultCalendarVersioningFormat, cfg.CalendarVersioningFormat(); e != g {
		t.Errorf("got: %s, expect: %s", g, e)
	}
}

func TestConfigCalendarVersioningFormatFromEnv(t *testing.T) {
	tmpdir := t.TempDir()
	confPath := filepath.Join(tmpdir, defaultConfigFile)

	t.Setenv("TAGPR_CALENDAR_VERSIONING", "YY.0M0D.MICRO")

	cfg := &config{
		conf:      confPath,
		gitconfig: &gitconfig.Config{GitPath: "git", File: confPath},
	}

	if err := cfg.Reload(); err != nil {
		t.Error(err)
	}

	if !cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be true from env")
	}
	if e, g := "YY.0M0D.MICRO", cfg.CalendarVersioningFormat(); e != g {
		t.Errorf("got: %s, expect: %s", g, e)
	}
}

func TestConfigCalendarVersioningRejectsMajorMinor(t *testing.T) {
	tmpdir := t.TempDir()
	confPath := filepath.Join(tmpdir, defaultConfigFile)
	cfg := &config{
		conf:      confPath,
		gitconfig: &gitconfig.Config{GitPath: "git", File: confPath},
	}

	if err := cfg.SetCalendarVersioning("YYYY.MAJOR.MICRO"); err == nil {
		t.Error("expected error for MAJOR token")
	}
	if err := cfg.SetCalendarVersioning("YYYY.MINOR.MICRO"); err == nil {
		t.Error("expected error for MINOR token")
	}
	if err := cfg.SetCalendarVersioning("YYYY.0M.MICRO"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigCalendarVersioningRejectsMajorMinorFromEnv(t *testing.T) {
	tmpdir := t.TempDir()
	confPath := filepath.Join(tmpdir, defaultConfigFile)

	t.Setenv("TAGPR_CALENDAR_VERSIONING", "YYYY.MAJOR.MICRO")

	cfg := &config{
		conf:      confPath,
		gitconfig: &gitconfig.Config{GitPath: "git", File: confPath},
	}

	if err := cfg.Reload(); err == nil {
		t.Error("expected error for MAJOR token in env")
	}
}

func TestFixedMajorVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    *uint64
		wantErr bool
	}{
		{"1", ptr(uint64(1)), false},
		{"10", ptr(uint64(10)), false},
		{"v1", ptr(uint64(1)), false},
		{"v10", ptr(uint64(10)), false},
		{"", nil, false},
		{"abc", nil, true},
		{"v", nil, true},
		{"-1", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg := &config{
				fixedMajorVersion: &tt.input,
			}
			got, err := cfg.FixedMajorVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("FixedMajorVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.want == nil {
				if got != nil {
					t.Errorf("FixedMajorVersion(%q) = %v, want nil", tt.input, *got)
				}
			} else {
				if got == nil {
					t.Errorf("FixedMajorVersion(%q) = nil, want %v", tt.input, *tt.want)
				} else if *got != *tt.want {
					t.Errorf("FixedMajorVersion(%q) = %v, want %v", tt.input, *got, *tt.want)
				}
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
