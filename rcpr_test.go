package rcpr

import (
	"testing"
)

func TestDetectVersionFile(t *testing.T) {
	f, err := detectVersionFile(".", "0.0.0")
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
