package tagpr

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunRejectsInvalidExecutionOptions(t *testing.T) {
	tests := map[string]struct {
		argv    []string
		wantErr string
	}{
		"invalid mode": {
			argv:    []string{"--mode", "invalid"},
			wantErr: "invalid mode",
		},
		"missing apply plan": {
			argv:    []string{"--mode", "apply"},
			wantErr: "--plan is required",
		},
		"plan in plan mode": {
			argv:    []string{"--mode", "plan", "--plan", "{}"},
			wantErr: "--plan is only valid in apply mode",
		},
		"plan in auto mode": {
			argv:    []string{"--plan", "{}"},
			wantErr: "--plan is only valid in apply mode",
		},
		"malformed plan": {
			argv:    []string{"--mode", "apply", "--plan", "{"},
			wantErr: "failed to parse --plan",
		},
		"null plan": {
			argv:    []string{"--mode", "apply", "--plan", "null"},
			wantErr: "--plan must be a JSON object",
		},
		"incomplete plan": {
			argv:    []string{"--mode", "apply", "--plan", `{"tag":"v1.2.3"}`},
			wantErr: "plan.commit_sha is required",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run(context.Background(), tt.argv, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Run() error = %v, want %q", err, tt.wantErr)
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
