package rcpr

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
