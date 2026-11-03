package tagpr

import (
	"strings"
	"testing"
)

func TestRunOptionsValidate(t *testing.T) {
	validPlan := releasePlan{
		Tag:                     "v1.2.3",
		CommitSHA:               strings.Repeat("a", 40),
		BaseTag:                 "v1.2.2",
		PullRequestNumber:       123,
		ReleaseNotesBoundarySHA: strings.Repeat("b", 40),
	}
	tests := map[string]struct {
		opts    runOptions
		wantErr bool
	}{
		"default auto": {
			opts: runOptions{Mode: executionModeAuto},
		},
		"plan": {
			opts: runOptions{Mode: executionModePlan},
		},
		"apply": {
			opts: runOptions{Mode: executionModeApply, Plan: &validPlan},
		},
		"invalid mode": {
			opts:    runOptions{Mode: "invalid"},
			wantErr: true,
		},
		"plan in auto": {
			opts:    runOptions{Mode: executionModeAuto, Plan: &validPlan},
			wantErr: true,
		},
		"plan in plan mode": {
			opts:    runOptions{Mode: executionModePlan, Plan: &releasePlan{}},
			wantErr: true,
		},
		"apply without plan": {
			opts:    runOptions{Mode: executionModeApply},
			wantErr: true,
		},
		"apply without tag": {
			opts: runOptions{
				Mode: executionModeApply,
				Plan: &releasePlan{
					CommitSHA:               validPlan.CommitSHA,
					ReleaseNotesBoundarySHA: validPlan.ReleaseNotesBoundarySHA,
					PullRequestNumber:       validPlan.PullRequestNumber,
				},
			},
			wantErr: true,
		},
		"apply without commit": {
			opts: runOptions{
				Mode: executionModeApply,
				Plan: &releasePlan{
					Tag:                     validPlan.Tag,
					ReleaseNotesBoundarySHA: validPlan.ReleaseNotesBoundarySHA,
					PullRequestNumber:       validPlan.PullRequestNumber,
				},
			},
			wantErr: true,
		},
		"apply without boundary": {
			opts: runOptions{
				Mode: executionModeApply,
				Plan: &releasePlan{
					Tag:               validPlan.Tag,
					CommitSHA:         validPlan.CommitSHA,
					PullRequestNumber: validPlan.PullRequestNumber,
				},
			},
			wantErr: true,
		},
		"apply without pull request": {
			opts: runOptions{
				Mode: executionModeApply,
				Plan: &releasePlan{
					Tag:                     validPlan.Tag,
					CommitSHA:               validPlan.CommitSHA,
					ReleaseNotesBoundarySHA: validPlan.ReleaseNotesBoundarySHA,
				},
			},
			wantErr: true,
		},
		"apply first release": {
			opts: runOptions{
				Mode: executionModeApply,
				Plan: &releasePlan{
					Tag:                     validPlan.Tag,
					CommitSHA:               validPlan.CommitSHA,
					PullRequestNumber:       validPlan.PullRequestNumber,
					ReleaseNotesBoundarySHA: validPlan.ReleaseNotesBoundarySHA,
				},
			},
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
