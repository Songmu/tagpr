package tagpr

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunRejectsInvalidExecutionOptions(t *testing.T) {
	tests := map[string][]string{
		"invalid mode": {"--mode", "invalid"},
		"missing tag candidate": {
			"--mode", "tag",
		},
		"candidate in prepare mode": {
			"--mode", "prepare",
			"--pending-tag", "v1.2.3",
		},
	}
	for name, argv := range tests {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run(context.Background(), argv, &stdout, &stderr)
			if err == nil {
				t.Fatal("Run() expected an error")
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}
	if !strings.HasPrefix(stdout.String(), "tagpr v") {
		t.Errorf("version output = %q", stdout.String())
	}
}
