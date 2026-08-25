package tagpr

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		if len(pulls) > 0 {
			return pulls[0], nil
		}
		if i < maxRetries-1 {
			log.Printf("ListPullRequestsWithCommit returned empty for %s, retrying in %s (%d/%d)",
				commitish, retryInterval, i+1, maxRetries)
			time.Sleep(retryInterval)
		}
	}
	return nil, nil
}

// mergedBaseSHA returns the SHA of the release branch tip just before the release
// pull request was merged. Since it is reported by the GitHub API as the base of the
// pull request, it is independent of the merge method, and it is correct for
// "Create a merge commit", "Squash and merge" and "Rebase and merge" alike.
// Note that "HEAD~" cannot be used for this purpose because it may point to a commit
// of the release pull request itself when "Rebase and merge" was used.
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

func (tp *tagpr) tagRelease(ctx context.Context, pr *github.PullRequest, currVer *semv, latestSemverTag string) error {
	var (
		vfile string
		err   error
	)
	releaseBranch := tp.cfg.ReleaseBranch()

	// The base SHA of the merged release pull request points to the release branch tip
	// just before the merge, regardless of which merge method was used.
	baseSHA, err := mergedBaseSHA(pr)
	if err != nil {
		return err
	}

	if tp.cfg.VersionFile() == "" {
		if _, _, err := tp.c.Git("checkout", baseSHA); err != nil {
			return err
		}
		vfile, err = detectVersionFile(".", currVer)
		if err != nil {
			return err
		}
		if _, _, err := tp.c.Git("checkout", releaseBranch); err != nil {
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
	// Use the base commit of the release pull request to avoid picking up the merge
	// of the pull request made by tagpr.
	targetCommitish := baseSHA
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
