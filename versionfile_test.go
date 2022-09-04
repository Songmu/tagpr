package tagpr

import (
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
	ver, err := retrieveVersionFromFile("version.go", false)
	if err != nil {
		t.Error(err)
	}
	if ver.Naked() != version {
		t.Errorf("detected: %s, expected: %s", ver.Naked(), version)
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
