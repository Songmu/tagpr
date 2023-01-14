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
	if err := cfg.SetRelaseBranch("main"); err != nil {
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

	b, err := os.ReadFile(confPath)
	if err != nil {
		t.Error(err)
	}

	var out string
	for _, line := range strings.Split(string(b), "\n") {
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
