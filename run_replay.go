package tagpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v93/github"
)

func (tp *tagpr) manualReleaseCommits(releaseBranch, rcBranch, commitMessage, changelogMessage string) []string {
	cherryLogArgs := tagLogArgs(tp.normalizedTagPrefix, "log", "--no-merges", "--pretty=format:%h %s",
		fmt.Sprintf("%s..%s/%s", releaseBranch, tp.remoteName, rcBranch))
	out, _, err := tp.c.Git(cherryLogArgs...)
	if err != nil {
		return nil
	}
	var cherryPicks []string
	for line := range strings.SplitSeq(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := strings.SplitN(line, " ", 2)
		if len(m) < 2 {
			continue
		}
		commitish := m[0]
		subject := strings.TrimSpace(m[1])
		if subject != commitMessage && subject != changelogMessage {
			cherryPicks = append(cherryPicks, commitish)
		}
	}
	return cherryPicks
}

func (tp *tagpr) replayManualReleaseCommits(ctx context.Context, releaseBranch, rcBranch, commitMessage, changelogMessage string, newCommit *github.Commit, ref *github.Reference) (*github.Commit, *github.Reference, error) {
	cherryPicks := tp.manualReleaseCommits(releaseBranch, rcBranch, commitMessage, changelogMessage)
	if len(cherryPicks) == 0 {
		return newCommit, ref, nil
	}
	tempRef := "refs/heads/tagpr-temp"
	resp, err := tp.gh.Git.DeleteRef(ctx, tp.owner, tp.repo, tempRef)
	if err != nil && resp.StatusCode != 422 {
		tp.showGHError(err, resp)
		return nil, nil, err
	}

	createRef := github.CreateRef{
		Ref: tempRef,
		SHA: *newCommit.SHA,
	}
	ref, resp, err = tp.gh.Git.CreateRef(ctx, tp.owner, tp.repo, createRef)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, nil, err
	}

	for i := len(cherryPicks) - 1; i >= 0; i-- {
		newCommit, err = tp.replayReleaseCommit(ctx, cherryPicks[i], tempRef, newCommit)
		if err != nil {
			return nil, nil, err
		}
	}
	// Checkout the temporary reference (Files like .tagpr used in subsequent processes may have
	// been rewritten during the cherry-pick process)
	if _, _, err := tp.c.Git("fetch"); err != nil {
		return nil, nil, err
	}
	if _, _, err := tp.c.Git("reset", "--hard"); err != nil {
		return nil, nil, err
	}
	if _, _, err := tp.c.Git("checkout", strings.TrimPrefix(tempRef, "refs/heads/")); err != nil {
		return nil, nil, err
	}

	resp, err = tp.gh.Git.DeleteRef(ctx, tp.owner, tp.repo, tempRef)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, nil, err
	}
	return newCommit, ref, nil
}

func (tp *tagpr) replayReleaseCommit(ctx context.Context, commitish, tempRef string, newCommit *github.Commit) (*github.Commit, error) {
	cherryPickCommit, resp, err := tp.gh.Repositories.GetCommit(
		ctx, tp.owner, tp.repo, commitish, nil)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}

	commit := github.Commit{
		Message: new("cherry-pick: " + commitish),
		Tree:    newCommit.Tree,
		Parents: cherryPickCommit.Parents,
	}
	tempCommit, resp, err := tp.gh.Git.CreateCommit(ctx, tp.owner, tp.repo, commit, nil)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}

	updateRef := github.UpdateRef{
		SHA:   *tempCommit.SHA,
		Force: new(true),
	}
	_, resp, err = tp.gh.Git.UpdateRef(ctx, tp.owner, tp.repo, tempRef, updateRef)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}

	mergeRequest := github.RepositoryMergeRequest{
		Base: strings.TrimPrefix(tempRef, "refs/heads/"),
		Head: commitish,
	}
	mergeCommit, resp, err := tp.gh.Repositories.Merge(
		ctx, tp.owner, tp.repo, mergeRequest)
	if err != nil {
		if resp.StatusCode == 409 {
			return newCommit, nil
		}
		tp.showGHError(err, resp)
		return nil, err
	}

	// The Author is not set because setting the same Author as the original commit makes it
	// difficult to create a Verified Commit.
	commit = github.Commit{
		Message: cherryPickCommit.Commit.Message,
		Tree:    mergeCommit.Commit.Tree,
		Parents: []*github.Commit{newCommit},
	}
	newCommit, resp, err = tp.gh.Git.CreateCommit(ctx, tp.owner, tp.repo, commit, nil)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}

	updateRef = github.UpdateRef{
		SHA:   *newCommit.SHA,
		Force: new(true),
	}
	_, resp, err = tp.gh.Git.UpdateRef(ctx, tp.owner, tp.repo, tempRef, updateRef)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}
	return newCommit, nil
}
