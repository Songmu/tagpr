package tagpr

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileOrder(t *testing.T) {
	input := []string{
		"aaa/ccc",
		"aaa.go",
		"bbb",
		"bb/ccd3",
		"l/m/n",
	}

	expect := []string{
		"aaa.go",
		"bbb",
		"aaa/ccc",
		"bb/ccd3",
		"l/m/n",
	}

	got := fileOrder(input)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("error: %v", got)
	}
}
func TestRetrieveVersionFile(t *testing.T) {
	ver, err := retrieveVersionFromFile("version.go", &semv{vPrefix: false})
	if err != nil {
		t.Error(err)
	}
	if ver.Naked() != version {
		t.Errorf("detected: %s, expected: %s", ver.Naked(), version)
	}

	ver, _ = retrieveVersionFromFile("testdata/vfile1", &semv{vPrefix: true})
	if e, g := "v1.2.3", ver.Tag(); e != g {
		t.Errorf("got: %s, expected: %s", g, e)
	}

	ver, _ = retrieveVersionFromFile("testdata/vfile2", &semv{vPrefix: false})
	if e, g := "1.3.5", ver.Tag(); e != g {
		t.Errorf("got: %s, expected: %s", g, e)
	}

	ver, _ = retrieveVersionFromFile("testdata/vfile3", &semv{vPrefix: false})
	if e, g := "12.3.4", ver.Tag(); e != g {
		t.Errorf("got: %s, expected: %s", g, e)
	}
}

func TestDetectVersionFile(t *testing.T) {
	v, _ := newSemver(version)
	f, err := detectVersionFile(".", v)
	if err != nil {
		t.Error(err)
	}
	if f != "version.go" {
		t.Errorf("error")
	}
}

func TestDetectVersionFile_perl(t *testing.T) {
	v, _ := newSemver("v1.0.0")
	f, err := detectVersionFile("testdata/perl", v)
	if err != nil {
		t.Errorf("error should be nil, but: %s", err)
	}
	if f != "lib/Riji.pm" {
		t.Errorf("error: %s", f)
	}
}

func TestDetectVersionFile_rust(t *testing.T) {
	v, _ := newSemver("v1.0.0")
	f, err := detectVersionFile("testdata/rust", v)
	if err != nil {
		t.Errorf("error should be nil, but: %s", err)
	}

	if f != "Cargo.toml" {
		t.Errorf("error: %s", f)
	}
}

func TestBumpVersionFile(t *testing.T) {
	from, err := newSemver("v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	to, err := newSemver("v1.3.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, input, want string
	}{
		{"first occurrence", `version = "1.2.3"; previous = "1.2.3"`, `version = "1.3.0"; previous = "1.2.3"`},
		{"preserve v prefix", `version = "v1.2.3"; previous = "1.2.3"`, `version = "v1.3.0"; previous = "1.2.3"`},
		{"no match", `version = "11.2.3"; other = "1.2.30"`, `version = "11.2.3"; other = "1.2.30"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "version")
			if err := os.WriteFile(path, []byte(tt.input), 0644); err != nil {
				t.Fatal(err)
			}
			if err := bumpVersionFile(path, from, to); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != tt.want {
				t.Errorf("bumped file = %q, want %q", content, tt.want)
			}
		})
	}
	t.Run("read failure", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing")
		if err := bumpVersionFile(path, from, to); !os.IsNotExist(err) {
			t.Errorf("bumpVersionFile() error = %v, want missing file", err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("missing version file was created: %v", err)
		}
	})
}
