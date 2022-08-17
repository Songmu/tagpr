package rcpr

import (
	"testing"
)

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

func TestRetrieveVersionFile(t *testing.T) {
	ver, err := retrieveVersionFromFile("version.go", false)
	if err != nil {
		t.Error(err)
	}
	t.Log(ver.Tag())
}
