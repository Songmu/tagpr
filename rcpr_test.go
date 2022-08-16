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
