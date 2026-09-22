package tagpr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v83/github"
)

func (tp *tagpr) mergedReleasePullRequestForCommit(
	ctx context.Context, commitish string,
) (*github.PullRequest, error) {
	// Retry because GitHub's internal commit-to-PR index may not be updated
	// immediately after a merge, causing the API to return an empty list.
	// This is especially common with squash merges but can also happen with
	// regular merge commits when the workflow triggers within seconds.
	// See https://github.com/Songmu/tagpr/issues/330
	const maxRetries = 3
	const retryInterval = 2 * time.Second

	for i := range maxRetries {
		opts := &github.ListOptions{PerPage: 100}
		foundAssociations := false
		for {
			pulls, resp, err := tp.gh.PullRequests.ListPullRequestsWithCommit(
				ctx, tp.owner, tp.repo, commitish, opts)
			if err != nil {
				showGHError(err, resp)
				return nil, err
			}
			foundAssociations = foundAssociations || len(pulls) > 0
			for _, pr := range pulls {
				if !pr.GetMergedAt().IsZero() &&
					pr.Base != nil &&
					pr.Base.GetRef() == tp.cfg.ReleaseBranch() &&
					tp.isTagPR(pr) {
					return pr, nil
				}
			}
			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
		if foundAssociations {
			return nil, nil
		}
		if i < maxRetries-1 {
			log.Printf("ListPullRequestsWithCommit returned empty for %s, retrying in %s (%d/%d)",
				commitish, retryInterval, i+1, maxRetries)
			time.Sleep(retryInterval)
		}
	}
	return nil, nil
}

func (tp *tagpr) latestMergedReleasePullRequest(ctx context.Context) (*github.PullRequest, error) {
	commitish, _, err := tp.c.Git("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	return tp.mergedReleasePullRequestForCommit(ctx, commitish)
}

const (
	envGitHubEventName = "GITHUB_EVENT_NAME"
	envGitHubEventPath = "GITHUB_EVENT_PATH"
)

// releaseBoundarySHA selects the pre-release boundary used to detect the version
// file and generate release notes without including the release pull request itself.
// Note that "HEAD~" cannot be used for this purpose because it may point to a commit
// of the release pull request itself when "Rebase and merge" was used.
func (tp *tagpr) releaseBoundarySHA(pr *github.PullRequest) (string, error) {
	if os.Getenv(envGitHubEventName) != "push" {
		return mergedBaseSHA(pr)
	}
	headSHA, _, err := tp.c.Git("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	sha, err := pushEventBeforeSHA(
		os.Getenv(envGitHubEventPath),
		"refs/heads/"+tp.cfg.ReleaseBranch(),
		headSHA,
	)
	if err != nil {
		return "", err
	}
	if sha != "" {
		return sha, nil
	}
	return mergedBaseSHA(pr)
}

// pushEventBeforeSHA returns the top-level "before" SHA of the push event payload.
// It returns an empty string when the payload is unavailable or has no usable
// "before", so that the caller can fall back to another source.
func pushEventBeforeSHA(path, releaseRef, headSHA string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read the event payload %q: %w", path, err)
	}

	var payload struct {
		Before string `json:"before"`
		After  string `json:"after"`
		Ref    string `json:"ref"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", fmt.Errorf("failed to parse the event payload %q: %w", path, err)
	}
	if payload.Ref != releaseRef || payload.After != headSHA {
		return "", nil
	}
	if isNullSHA(payload.Before) {
		return "", nil
	}
	return payload.Before, nil
}

// isNullSHA reports whether the SHA is empty or the all-zero SHA, which Git and
// GitHub use to represent "no commit".
func isNullSHA(sha string) bool {
	return sha == "" || strings.Trim(sha, "0") == ""
}

// mergedBaseSHA returns the base SHA of the merged release pull request. It is used
// as a fallback when the push event payload is unavailable. Note that it is the base
// at the time the pull request was last updated, so it may be stale if the release
// branch advanced before the merge. In that case some changes may be missing from
// the release notes, which is acceptable because tagpr is expected to refresh the
// release pull request before it is merged.
func mergedBaseSHA(pr *github.PullRequest) (string, error) {
	if pr == nil {
		return "", errors.New("failed to detect the base commit of the release pull request: no pull request")
	}
	if pr.Base == nil {
		return "", fmt.Errorf(
			"failed to detect the base commit of the release pull request #%d: no base", pr.GetNumber())
	}
	sha := pr.Base.GetSHA()
	if sha == "" {
		return "", fmt.Errorf(
			"failed to detect the base commit of the release pull request #%d: empty base SHA", pr.GetNumber())
	}
	return sha, nil
}

func (tp *tagpr) withCheckout(commitish, restoreBranch string, fn func() error) (err error) {
	if _, _, err := tp.c.Git("checkout", commitish); err != nil {
		return err
	}
	defer func() {
		if _, _, restoreErr := tp.c.Git("checkout", restoreBranch); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to checkout %s: %w", restoreBranch, restoreErr))
		}
	}()
	return fn()
}

func (tp *tagpr) prepareReleaseCandidate(
	pr *github.PullRequest, currVer *semv, latestSemverTag string,
) (releaseCandidate, error) {
	targetSHA, _, err := tp.c.Git("rev-parse", "HEAD")
	if err != nil {
		return releaseCandidate{}, err
	}
	boundarySHA, err := tp.releaseBoundarySHA(pr)
	if err != nil {
		return releaseCandidate{}, err
	}
	pendingTag, err := tp.calculatePendingTag(
		pr, currVer, boundarySHA, targetSHA, tp.cfg.ReleaseBranch())
	if err != nil {
		return releaseCandidate{}, err
	}
	return releaseCandidate{
		PendingTag:         pendingTag,
		TargetSHA:          targetSHA,
		ReleaseBoundarySHA: boundarySHA,
		PullRequestNumber:  pr.GetNumber(),
		BaseTag:            latestSemverTag,
	}, nil
}

func (tp *tagpr) calculatePendingTag(
	pr *github.PullRequest,
	currVer *semv,
	boundarySHA, targetSHA, restoreCommitish string,
) (string, error) {
	var vfile string
	if tp.cfg.VersionFile() == "" {
		if err := tp.withCheckout(boundarySHA, restoreCommitish, func() error {
			var detectErr error
			vfile, detectErr = detectVersionFile(".", currVer)
			return detectErr
		}); err != nil {
			return "", err
		}
	} else if tp.cfg.VersionFile() != "-" {
		vfiles := strings.Split(tp.cfg.VersionFile(), ",")
		vfile = strings.TrimSpace(vfiles[0])
	}

	var nextTag string
	if vfile != "" {
		if err := tp.withCheckout(targetSHA, restoreCommitish, func() error {
			nextVer, retrieveErr := retrieveVersionFromFile(vfile, currVer)
			if retrieveErr != nil {
				return retrieveErr
			}
			nextTag = nextVer.Tag()
			return nil
		}); err != nil {
			return "", err
		}
	} else {
		var labels []string
		for _, l := range pr.Labels {
			labels = append(labels, l.GetName())
		}
		nextTag = currVer.GuessNext(labels).Tag()
	}
	return fullTag(tp.normalizedTagPrefix, nextTag), nil
}

func (tp *tagpr) setCandidateOutputs(candidate releaseCandidate) error {
	outputs := []struct {
		name  string
		value string
	}{
		{"pending_tag", candidate.PendingTag},
		{"target_sha", candidate.TargetSHA},
		{"release_boundary_sha", candidate.ReleaseBoundarySHA},
		{"pull_request_number", fmt.Sprintf("%d", candidate.PullRequestNumber)},
	}
	for _, output := range outputs {
		if err := tp.setOutput(output.name, output.value); err != nil {
			return err
		}
	}
	return nil
}

func (tp *tagpr) finalizeRelease(ctx context.Context, candidate releaseCandidate) error {
	if err := (runOptions{Mode: executionModeTag, Candidate: candidate}).validate(); err != nil {
		return err
	}
	targetSHA, err := tp.resolveExactCommit(candidate.TargetSHA)
	if err != nil {
		return fmt.Errorf("failed to resolve target SHA: %w", err)
	}
	boundarySHA, err := tp.resolveExactCommit(candidate.ReleaseBoundarySHA)
	if err != nil {
		return fmt.Errorf("failed to resolve release boundary SHA: %w", err)
	}
	candidate.TargetSHA = targetSHA
	candidate.ReleaseBoundarySHA = boundarySHA

	if _, _, err := tp.c.Git(
		"fetch", tp.remote(),
		"+refs/heads/"+tp.cfg.ReleaseBranch()+
			":refs/remotes/"+tp.remote()+"/"+tp.cfg.ReleaseBranch(),
	); err != nil {
		return fmt.Errorf("failed to fetch release branch: %w", err)
	}
	remoteReleaseBranch := tp.remote() + "/" + tp.cfg.ReleaseBranch()
	if err := tp.requireAncestor(targetSHA, remoteReleaseBranch,
		"target commit is not an ancestor of the release branch"); err != nil {
		return err
	}
	if err := tp.requireAncestor(boundarySHA, targetSHA,
		"release boundary is not an ancestor of the target commit"); err != nil {
		return err
	}
	if candidate.BaseTag != "" {
		if err := tp.requireAncestor(candidate.BaseTag, boundarySHA,
			"base tag is not an ancestor of the release boundary"); err != nil {
			return err
		}
	}

	pr, err := tp.mergedReleasePullRequestForCommit(ctx, targetSHA)
	if err != nil {
		return err
	}
	if pr == nil || pr.GetNumber() != candidate.PullRequestNumber {
		return fmt.Errorf(
			"target commit %s is not associated with merged tagpr pull request #%d",
			targetSHA, candidate.PullRequestNumber)
	}

	localTagExists, remoteTagExists, err := tp.inspectExistingTag(
		candidate.PendingTag, targetSHA)
	if err != nil {
		return err
	}
	latestTag := tp.latestSemverTag()
	if latestTag != candidate.BaseTag &&
		!((localTagExists || remoteTagExists) && latestTag == candidate.PendingTag) {
		return fmt.Errorf("base tag changed from %q to %q", candidate.BaseTag, latestTag)
	}

	currVer, err := tp.versionFromBaseTag(candidate.BaseTag)
	if err != nil {
		return err
	}
	recalculatedTag, err := tp.calculatePendingTag(
		pr, currVer, boundarySHA, targetSHA, targetSHA)
	if err != nil {
		return err
	}
	if recalculatedTag != candidate.PendingTag {
		return fmt.Errorf(
			"pending tag changed from %q to %q", candidate.PendingTag, recalculatedTag)
	}
	return tp.completeRelease(ctx, candidate, pr, remoteTagExists)
}

func (tp *tagpr) completeRelease(
	ctx context.Context,
	candidate releaseCandidate,
	pr *github.PullRequest,
	tagExists bool,
) error {
	previousTag := &candidate.BaseTag
	if candidate.BaseTag == "" {
		previousTag = nil
	}
	targetCommitish := candidate.ReleaseBoundarySHA
	releases, resp, err := tp.gh.Repositories.GenerateReleaseNotes(
		ctx, tp.owner, tp.repo, &github.GenerateNotesOptions{
			TagName:               candidate.PendingTag,
			PreviousTagName:       previousTag,
			TargetCommitish:       &targetCommitish,
			ConfigurationFilePath: github.Ptr(tp.cfg.ReleaseYAMLPath()),
		})
	if err != nil {
		showGHError(err, resp)
		return err
	}

	if !tagExists {
		localSHA, _, localErr := tp.c.Git(
			"rev-parse", "--verify", "refs/tags/"+candidate.PendingTag+"^{commit}")
		if localErr != nil {
			if _, _, err := tp.c.Git(
				"tag", candidate.PendingTag, candidate.TargetSHA); err != nil {
				return err
			}
		} else if localSHA != candidate.TargetSHA {
			return fmt.Errorf(
				"tag %s already points to %s, want %s",
				candidate.PendingTag, localSHA, candidate.TargetSHA)
		}
		ref := "refs/tags/" + candidate.PendingTag
		if _, _, err := tp.c.Git("push", tp.remote(), ref+":"+ref); err != nil {
			_, existsAfterPush, verifyErr := tp.inspectExistingTag(
				candidate.PendingTag, candidate.TargetSHA)
			if verifyErr != nil || !existsAfterPush {
				return err
			}
		}
	}

	if !tp.cfg.Release() {
		return tp.setFinalOutputs(candidate, pr)
	}
	existingRelease, resp, err := tp.gh.Repositories.GetReleaseByTag(
		ctx, tp.owner, tp.repo, candidate.PendingTag)
	if err == nil {
		if existingRelease.GetDraft() != tp.cfg.ReleaseDraft() {
			return fmt.Errorf(
				"GitHub Release for %s already exists with draft=%t, want draft=%t",
				candidate.PendingTag, existingRelease.GetDraft(), tp.cfg.ReleaseDraft())
		}
		return tp.setFinalOutputs(candidate, pr)
	}
	if resp == nil || resp.StatusCode != 404 {
		showGHError(err, resp)
		return err
	}
	_, resp, err = tp.gh.Repositories.CreateRelease(
		ctx, tp.owner, tp.repo, &github.RepositoryRelease{
			TagName:         &candidate.PendingTag,
			TargetCommitish: &candidate.TargetSHA,
			Name:            &releases.Name,
			Body:            &releases.Body,
			Draft:           github.Ptr(tp.cfg.ReleaseDraft()),
		})
	if err != nil {
		showGHError(err, resp)
		return err
	}
	return tp.setFinalOutputs(candidate, pr)
}

func (tp *tagpr) setFinalOutputs(candidate releaseCandidate, pr *github.PullRequest) error {
	if err := tp.setOutput("tag", candidate.PendingTag); err != nil {
		return err
	}
	if err := tp.setOutput("base_tag", candidate.BaseTag); err != nil {
		return err
	}
	b, err := json.Marshal(pr)
	if err != nil {
		return err
	}
	return tp.setOutput("pull_request", string(b))
}

func (tp *tagpr) resolveExactCommit(candidateSHA string) (string, error) {
	sha, _, err := tp.c.Git("rev-parse", "--verify", candidateSHA+"^{commit}")
	if err != nil {
		return "", err
	}
	if sha != candidateSHA {
		return "", fmt.Errorf("%q is not a full commit SHA", candidateSHA)
	}
	return sha, nil
}

func (tp *tagpr) requireAncestor(ancestor, descendant, message string) error {
	if _, _, err := tp.c.Git("merge-base", "--is-ancestor", ancestor, descendant); err != nil {
		return fmt.Errorf("%s: %s is not an ancestor of %s", message, ancestor, descendant)
	}
	return nil
}

func (tp *tagpr) inspectExistingTag(tag, targetSHA string) (bool, bool, error) {
	localSHA, _, localErr := tp.c.Git("rev-parse", "--verify", "refs/tags/"+tag+"^{commit}")
	if localErr == nil && localSHA != targetSHA {
		return false, false,
			fmt.Errorf("tag %s already points to %s, want %s", tag, localSHA, targetSHA)
	}

	out, _, err := tp.c.Git("ls-remote", "--tags", tp.remote(), "refs/tags/"+tag)
	if err != nil {
		return false, false, err
	}
	if out == "" {
		return localErr == nil, false, nil
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return false, false, fmt.Errorf("failed to parse remote tag %s", tag)
	}
	if fields[0] != targetSHA {
		return false, false, fmt.Errorf(
			"remote tag %s already points to %s, want %s", tag, fields[0], targetSHA)
	}
	return localErr == nil, true, nil
}

func (tp *tagpr) remote() string {
	if tp.remoteName == "" {
		return "origin"
	}
	return tp.remoteName
}

func (tp *tagpr) versionFromBaseTag(baseTag string) (*semv, error) {
	version := baseTag
	if version == "" {
		version = "v0.0.0"
	} else {
		version = strings.TrimPrefix(version, tp.normalizedTagPrefix)
	}
	currVer, err := newSemver(version)
	if err != nil {
		return nil, err
	}
	if tp.cfg.vPrefix == nil {
		currVer.vPrefix = strings.HasPrefix(version, "v")
	} else {
		currVer.vPrefix = *tp.cfg.vPrefix
	}
	currVer.asCalendarVersion = tp.cfg.CalendarVersioning()
	currVer.calverFormat = tp.cfg.CalendarVersioningFormat()
	return currVer, nil
}

func (tp *tagpr) tagRelease(
	ctx context.Context, pr *github.PullRequest, currVer *semv, latestSemverTag string,
) error {
	candidate, err := tp.prepareReleaseCandidate(pr, currVer, latestSemverTag)
	if err != nil {
		return err
	}
	return tp.completeRelease(ctx, candidate, pr, false)
}
