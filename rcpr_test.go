package rcpr

import (
	"testing"
)

func TestDetectVersionFile(t *testing.T) {
	v, _ := newSemver("0.0.0")
	f, err := detectVersionFile(".", v)
	if err != nil {
		t.Error(err)
	}
	t.Log(f)
}

func TestRetrieveVersionFile(t *testing.T) {
	ver, err := retrieveVersionFromFile("version.go", false)
	if err != nil {
		t.Error(err)
	}
	t.Log(ver.Tag())
}
