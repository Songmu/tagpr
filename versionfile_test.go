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
	ver, err := retrieveVersionFromFile("version.go", false, nil)
	if err != nil {
		t.Error(err)
	}
	if ver.Naked() != version {
		t.Errorf("detected: %s, expected: %s", ver.Naked(), version)
	}

	ver, _ = retrieveVersionFromFile("testdata/vfile1", true, nil)
	if e, g := "v1.2.3", ver.Tag(); e != g {
		t.Errorf("got: %s, expected: %s", g, e)
	}

	ver, _ = retrieveVersionFromFile("testdata/vfile2", false, nil)
	if e, g := "1.3.5", ver.Tag(); e != g {
		t.Errorf("got: %s, expected: %s", g, e)
	}

	ver, _ = retrieveVersionFromFile("testdata/vfile3", false, nil)
	if e, g := "12.3.4", ver.Tag(); e != g {
		t.Errorf("got: %s, expected: %s", g, e)
	}
}

func TestDetectVersionFile(t *testing.T) {
	v, _ := newSemver(version, nil)
	f, err := detectVersionFile(".", v)
	if err != nil {
		t.Error(err)
	}
	if f != "version.go" {
		t.Errorf("error")
	}
}

func TestDetectVersionFile_perl(t *testing.T) {
	v, _ := newSemver("v1.0.0", nil)
	f, err := detectVersionFile("testdata/perl", v)
	if err != nil {
		t.Errorf("error should be nil, but: %s", err)
	}
	if f != "lib/Riji.pm" {
		t.Errorf("error: %s", f)
	}
}
