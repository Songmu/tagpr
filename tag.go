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

func (tp *tagpr) latestPullRequest(ctx context.Context) (*github.PullRequest, error) {
	// tag and exit if the HEAD is the merged tagpr
	commitish, _, err := tp.c.Git("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}

	// Retry because GitHub's internal commit-to-PR index may not be updated
	// immediately after a merge, causing the API to return an empty list.
	// This is especially common with squash merges but can also happen with
	// regular merge commits when the workflow triggers within seconds.
	// See https://github.com/Songmu/tagpr/issues/330
	const maxRetries = 3
	const retryInterval = 2 * time.Second

	for i := range maxRetries {
		pulls, resp, err := tp.gh.PullRequests.ListPullRequestsWithCommit(
			ctx, tp.owner, tp.repo, commitish, nil)
		if err != nil {
			showGHError(err, resp)
			return nil, err
		}
		for _, pr := range pulls {
			if !pr.GetMergedAt().IsZero() && tp.isTagPR(pr) {
				return pr, nil
			}
		}
		if len(pulls) > 0 {
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

const (
	envGitHubEventName = "GITHUB_EVENT_NAME"
	envGitHubEventPath = "GITHUB_EVENT_PATH"
)

// releaseBoundarySHA selects the pre-release boundary used to detect the version
// file and generate release notes without including the release pull request itself.
// Note that "HEAD~" cannot be used for this purpose because it may point to a commit
// of the release pull request itself when "Rebase and merge" was used.
func releaseBoundarySHA(pr *github.PullRequest) (string, error) {
	if os.Getenv(envGitHubEventName) != "push" {
		return mergedBaseSHA(pr)
	}
	sha, err := pushEventBeforeSHA(os.Getenv(envGitHubEventPath))
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
func pushEventBeforeSHA(path string) (string, error) {
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
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", fmt.Errorf("failed to parse the event payload %q: %w", path, err)
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

func (tp *tagpr) tagRelease(ctx context.Context, pr *github.PullRequest, currVer *semv, latestSemverTag string) error {
	var (
		vfile string
		err   error
	)
	releaseBranch := tp.cfg.ReleaseBranch()

	boundarySHA, err := releaseBoundarySHA(pr)
	if err != nil {
		return err
	}

	if tp.cfg.VersionFile() == "" {
		if err := tp.withCheckout(boundarySHA, releaseBranch, func() error {
			var detectErr error
			vfile, detectErr = detectVersionFile(".", currVer)
			return detectErr
		}); err != nil {
			return err
		}
	} else if tp.cfg.VersionFile() != "-" {
		vfiles := strings.Split(tp.cfg.VersionFile(), ",")
		vfile = strings.TrimSpace(vfiles[0])
	}

	var nextTag string
	if vfile != "" {
		nextVer, err := retrieveVersionFromFile(vfile, currVer)
		if err != nil {
			return err
		}
		nextTag = nextVer.Tag()
	} else {
		var labels []string
		for _, l := range pr.Labels {
			labels = append(labels, l.GetName())
		}
		nextTag = currVer.GuessNext(labels).Tag()
	}
	// Add prefix for monorepo support
	fullNextTag := fullTag(tp.normalizedTagPrefix, nextTag)

	previousTag := &latestSemverTag
	if *previousTag == "" {
		previousTag = nil
	}

	// To avoid putting pull requests created by tagpr itself in the release notes,
	// we generate release notes in advance.
	// Stop at the selected boundary to exclude the release pull request itself.
	targetCommitish := boundarySHA
	releases, resp, err := tp.gh.Repositories.GenerateReleaseNotes(
		ctx, tp.owner, tp.repo, &github.GenerateNotesOptions{
			TagName:               fullNextTag,
			PreviousTagName:       previousTag,
			TargetCommitish:       &targetCommitish,
			ConfigurationFilePath: github.Ptr(tp.cfg.ReleaseYAMLPath()),
		})
	if err != nil {
		showGHError(err, resp)
		return err
	}

	if _, _, err := tp.c.Git("tag", fullNextTag); err != nil {
		return err
	}
	_, _, err = tp.c.Git("push", "--tags")
	if err != nil {
		return err
	}
	tp.setOutput("tag", fullNextTag)

	if !tp.cfg.Release() {
		return nil
	}
	// Don't use GenerateReleaseNote flag and use pre generated one
	_, resp, err = tp.gh.Repositories.CreateRelease(
		ctx, tp.owner, tp.repo, &github.RepositoryRelease{
			TagName:         &fullNextTag,
			TargetCommitish: &releaseBranch,
			Name:            &releases.Name,
			Body:            &releases.Body,
			Draft:           github.Ptr(tp.cfg.ReleaseDraft()),
		})
	if err != nil {
		showGHError(err, resp)
		return err
	}
	return nil
}
