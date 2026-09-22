package tagpr

import (
	"strings"
	"testing"
)

func TestRunOptionsValidate(t *testing.T) {
	validCandidate := releaseCandidate{
		PendingTag:         "v1.2.3",
		TargetSHA:          strings.Repeat("a", 40),
		ReleaseBoundarySHA: strings.Repeat("b", 40),
		PullRequestNumber:  123,
		BaseTag:            "v1.2.2",
	}
	tests := map[string]struct {
		opts    runOptions
		wantErr bool
	}{
		"default auto": {
			opts: runOptions{Mode: executionModeAuto},
		},
		"prepare": {
			opts: runOptions{Mode: executionModePrepare},
		},
		"tag": {
			opts: runOptions{Mode: executionModeTag, Candidate: validCandidate},
		},
		"invalid mode": {
			opts:    runOptions{Mode: "invalid"},
			wantErr: true,
		},
		"candidate in auto": {
			opts:    runOptions{Mode: executionModeAuto, Candidate: validCandidate},
			wantErr: true,
		},
		"candidate in prepare": {
			opts:    runOptions{Mode: executionModePrepare, Candidate: validCandidate},
			wantErr: true,
		},
		"tag without pending tag": {
			opts: runOptions{
				Mode: executionModeTag,
				Candidate: releaseCandidate{
					TargetSHA:          validCandidate.TargetSHA,
					ReleaseBoundarySHA: validCandidate.ReleaseBoundarySHA,
					PullRequestNumber:  validCandidate.PullRequestNumber,
				},
			},
			wantErr: true,
		},
		"tag without target": {
			opts: runOptions{
				Mode: executionModeTag,
				Candidate: releaseCandidate{
					PendingTag:         validCandidate.PendingTag,
					ReleaseBoundarySHA: validCandidate.ReleaseBoundarySHA,
					PullRequestNumber:  validCandidate.PullRequestNumber,
				},
			},
			wantErr: true,
		},
		"tag without boundary": {
			opts: runOptions{
				Mode: executionModeTag,
				Candidate: releaseCandidate{
					PendingTag:        validCandidate.PendingTag,
					TargetSHA:         validCandidate.TargetSHA,
					PullRequestNumber: validCandidate.PullRequestNumber,
				},
			},
			wantErr: true,
		},
		"tag without pull request": {
			opts: runOptions{
				Mode: executionModeTag,
				Candidate: releaseCandidate{
					PendingTag:         validCandidate.PendingTag,
					TargetSHA:          validCandidate.TargetSHA,
					ReleaseBoundarySHA: validCandidate.ReleaseBoundarySHA,
				},
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.opts.validate()
			if tt.wantErr && err == nil {
				t.Fatal("validate() expected an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validate() failed: %v", err)
			}
		})
	}
}
