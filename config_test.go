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

	// Initially false (not set)
	if cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be false initially")
	}

	// Set to true
	if err := cfg.SetCalendarVersioning(true); err != nil {
		t.Error(err)
	}
	if !cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be true")
	}

	// Reload and check persistence
	if err := cfg.Reload(); err != nil {
		t.Error(err)
	}
	if !cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be true after reload")
	}

	// Set to false
	if err := cfg.SetCalendarVersioning(false); err != nil {
		t.Error(err)
	}
	if cfg.CalendarVersioning() {
		t.Error("CalendarVersioning should be false")
	}
}

func TestConfigCalendarVersioningFromEnv(t *testing.T) {
	tmpdir := t.TempDir()
	confPath := filepath.Join(tmpdir, defaultConfigFile)

	// Set environment variable
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
}
