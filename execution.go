package tagpr

import "fmt"

type executionMode string

const (
	executionModeAuto    executionMode = "auto"
	executionModePrepare executionMode = "prepare"
	executionModeTag     executionMode = "tag"
)

type releaseCandidate struct {
	PendingTag         string
	TargetSHA          string
	ReleaseBoundarySHA string
	PullRequestNumber  int
	BaseTag            string
}

type runOptions struct {
	Mode      executionMode
	Candidate releaseCandidate
}

func (opts runOptions) validate() error {
	switch opts.Mode {
	case executionModeAuto, executionModePrepare:
		if opts.Candidate != (releaseCandidate{}) {
			return fmt.Errorf("release candidate options are only valid in tag mode")
		}
	case executionModeTag:
		if opts.Candidate.PendingTag == "" {
			return fmt.Errorf("--pending-tag is required in tag mode")
		}
		if opts.Candidate.TargetSHA == "" {
			return fmt.Errorf("--target-sha is required in tag mode")
		}
		if opts.Candidate.ReleaseBoundarySHA == "" {
			return fmt.Errorf("--release-boundary-sha is required in tag mode")
		}
		if opts.Candidate.PullRequestNumber <= 0 {
			return fmt.Errorf("--pull-request-number must be greater than zero in tag mode")
		}
	default:
		return fmt.Errorf("invalid mode %q: expected auto, prepare, or tag", opts.Mode)
	}
	return nil
}
