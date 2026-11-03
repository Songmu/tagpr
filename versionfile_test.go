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

func TestVersionFile(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		file     string
		language string
	}{
		{name: "empty"},
		{"fallback uses depth then name", []string{"a/version", "z.txt", "b.txt"}, "b.txt", ""},
		{"gemspec by name order", []string{"version.go", "app.gemspec"}, "app.gemspec", "ruby"},
		{"go by depth", []string{"lib/app.gemspec", "version.go"}, "version.go", "go"},
		{"setup cfg by case-sensitive name order", []string{"package.json", "SETUP.cfg"}, "SETUP.cfg", "python"},
		{"setup py by depth", []string{"src/version.go", "setup.py"}, "setup.py", "python"},
		{"package by case-sensitive name order", []string{"manifest.json", "Package.json"}, "Package.json", "node"},
		{"manifest by name order", []string{"package.json", "manifest.json"}, "manifest.json", "javascript"},
		{"pom by depth", []string{"src/package.json", "POM.XML"}, "POM.XML", "java"},
		{"cargo by case-sensitive name order", []string{"app.gemspec", "Cargo.toml"}, "Cargo.toml", "rust"},
		{"meta alone", []string{"META.JSON"}, "META.JSON", "perl"},
		{"module after meta", []string{"lib/App.pm", "meta.json"}, "lib/App.pm", "perl"},
		{"module before meta is not revisited", []string{"lib/App.pm", "lib/meta.json"}, "lib/meta.json", "perl"},
		{"module without meta", []string{"lib/App.pm", "other.txt"}, "other.txt", ""},
		{"first meta retained", []string{"nested/meta.json", "meta.json"}, "meta.json", "perl"},
		{"go before meta fallback", []string{"meta.json", "version.go"}, "version.go", "go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, language := versionFile(tt.files)
			if file != tt.file || language != tt.language {
				t.Errorf("got: (%q, %q), expected: (%q, %q)", file, language, tt.file, tt.language)
			}
		})
	}
}

func TestDetectVersionFile(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		version string
		file    string
	}{
		{"go", ".", version, "version.go"},
		{"perl", "testdata/perl", "v1.0.0", "lib/Riji.pm"},
		{"rust", "testdata/rust", "v1.0.0", "Cargo.toml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, _ := newSemver(tt.version)
			f, err := detectVersionFile(tt.root, v)
			if err != nil {
				t.Fatal(err)
			}
			if f != tt.file {
				t.Errorf("got: %s, expected: %s", f, tt.file)
			}
		})
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
