package tagpr

import "fmt"

type executionMode string

const (
	executionModeAuto  executionMode = "auto"
	executionModePlan  executionMode = "plan"
	executionModeApply executionMode = "apply"
)

type releasePlan struct {
	Tag                     string `json:"tag"`
	CommitSHA               string `json:"commit_sha"`
	BaseTag                 string `json:"base_tag"`
	PullRequestNumber       int    `json:"pull_request_number"`
	ReleaseNotesBoundarySHA string `json:"release_notes_boundary_sha"`
}

type runOptions struct {
	Mode executionMode
	Plan *releasePlan
}

func (opts runOptions) validate() error {
	switch opts.Mode {
	case executionModeAuto, executionModePlan:
		if opts.Plan != nil {
			return fmt.Errorf("--plan is only valid in apply mode")
		}
	case executionModeApply:
		if opts.Plan == nil {
			return fmt.Errorf("--plan is required in apply mode")
		}
		return opts.Plan.validate()
	default:
		return fmt.Errorf("invalid mode %q: expected auto, plan, or apply", opts.Mode)
	}
	return nil
}

func (plan releasePlan) validate() error {
	if plan.Tag == "" {
		return fmt.Errorf("plan.tag is required")
	}
	if plan.CommitSHA == "" {
		return fmt.Errorf("plan.commit_sha is required")
	}
	if plan.ReleaseNotesBoundarySHA == "" {
		return fmt.Errorf("plan.release_notes_boundary_sha is required")
	}
	if plan.PullRequestNumber <= 0 {
		return fmt.Errorf("plan.pull_request_number must be greater than zero")
	}
	return nil
}
