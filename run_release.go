package tagpr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-github/v93/github"
)

func (tp *tagpr) configureReleaseBranch() (string, error) {
	releaseBranch := tp.cfg.ReleaseBranch()
	if releaseBranch == "" {
		releaseBranch, _ = tp.defaultBranch()
		if releaseBranch == "" {
			releaseBranch = defaultReleaseBranch
		}
		if err := tp.cfg.SetReleaseBranch(releaseBranch); err != nil {
			return "", err
		}
	}
	return releaseBranch, nil
}

func (tp *tagpr) currentReleaseVersion() (string, string, *semv, error) {
	latestSemverTag := tp.latestSemverTag()
	tp.setOutput("base_tag", latestSemverTag)
	currVerStr := latestSemverTag
	fromCommitish := "refs/tags/" + currVerStr
	if currVerStr == "" {
		var err error
		fromCommitish, _, err = tp.c.Git("rev-list", "--max-parents=0", "HEAD")
		if err != nil {
			return "", "", nil, err
		}
		currVerStr = "v0.0.0"
	} else {
		// Strip prefix for newSemver (fromCommitish already has full tag name)
		currVerStr = strings.TrimPrefix(currVerStr, tp.normalizedTagPrefix)
	}
	currVer, err := newSemver(currVerStr)
	if err != nil {
		return "", "", nil, err
	}

	if tp.cfg.vPrefix == nil {
		if err := tp.cfg.SetVPrefix(currVer.vPrefix); err != nil {
			return "", "", nil, err
		}
	} else {
		currVer.vPrefix = *tp.cfg.vPrefix
	}

	currVer.asCalendarVersion = tp.cfg.CalendarVersioning()
	currVer.calverFormat = tp.cfg.CalendarVersioningFormat()

	return latestSemverTag, fromCommitish, currVer, nil
}

func (tp *tagpr) requireReleaseBranch(releaseBranch string) error {
	branch, _, err := tp.c.Git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to git symbolic-ref: %w", err)
	}
	if branch != releaseBranch {
		return fmt.Errorf("you are not on release branch %q, current branch is %q",
			releaseBranch, branch)
	}

	return nil
}

func (tp *tagpr) prepareMergedReleasePlan(pr *github.PullRequest) (releasePlan, error) {
	if tp.runOptions.Mode != executionModePlan {
		if err := tp.fetchReleaseRefs(); err != nil {
			return releasePlan{}, fmt.Errorf("failed to fetch release tags: %w", err)
		}
	}
	latestSemverTag, err := tp.releasePullRequestBaseTag(pr)
	if err != nil {
		return releasePlan{}, err
	}
	currVer, err := tp.versionFromBaseTag(latestSemverTag)
	if err != nil {
		return releasePlan{}, err
	}
	plan, err := tp.prepareReleasePlan(pr, currVer, latestSemverTag)
	if err != nil {
		return releasePlan{}, err
	}
	if tp.cfg.CalendarVersioning() {
		if err := tp.reuseCalendarReleaseTag(&plan, currVer); err != nil {
			return releasePlan{}, err
		}
	}
	if plan.BaseTag != "" {
		if err := tp.requireAncestor(plan.BaseTag, plan.ReleaseNotesBoundarySHA,
			"base tag is not an ancestor of the release boundary"); err != nil {
			return releasePlan{}, err
		}
	}
	return plan, nil
}

func (tp *tagpr) reuseCalendarReleaseTag(plan *releasePlan, currVer *semv) error {
	vfile, err := tp.releaseVersionFile(currVer, plan.ReleaseNotesBoundarySHA, plan.CommitSHA)
	if err != nil {
		return err
	}
	if vfile == "" {
		latestTag := tp.latestSemverTag()
		if latestTag != "" && latestTag != plan.BaseTag {
			sha, _, err := tp.c.Git("rev-parse", "--verify", "refs/tags/"+latestTag+"^{commit}")
			if err != nil {
				return err
			}
			if sha == plan.CommitSHA {
				plan.Tag = latestTag
			}
		}
	}
	return nil
}

func (tp *tagpr) handleMergedRelease(ctx context.Context, pr *github.PullRequest) error {
	plan, err := tp.prepareMergedReleasePlan(pr)
	if err != nil {
		return err
	}
	if tp.runOptions.Mode == executionModePlan {
		if err := tp.setOutput("base_tag", plan.BaseTag); err != nil {
			return err
		}
		if err := tp.setPlanOutput(plan); err != nil {
			return err
		}
		b, _ := json.Marshal(pr)
		tp.setOutput("pull_request", string(b))
		return nil
	}
	latestTag := tp.latestSemverTag(plan.Tag)
	if latestTag != plan.BaseTag {
		return fmt.Errorf("base tag changed from %q to %q", plan.BaseTag, latestTag)
	}
	_, tagExists, err := tp.inspectExistingTag(plan.Tag, plan.CommitSHA)
	if err != nil {
		return err
	}
	return tp.completeRelease(ctx, plan, pr, tagExists)
}
