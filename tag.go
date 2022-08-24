package rcpr

import (
	"context"

	"github.com/google/go-github/v45/github"
)

func (rp *rcpr) latestPullRequest(ctx context.Context) (*github.PullRequest, error) {
	// tag and exit if the HEAD is the merged rcpr
	commitish, _, err := rp.c.GitE("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	pulls, _, err := rp.gh.PullRequests.ListPullRequestsWithCommit(
		ctx, rp.owner, rp.repo, commitish, nil)
	if err != nil {
		return nil, err
	}
	if len(pulls) == 0 {
		return nil, nil
	}
	return pulls[0], nil
}

func (rp *rcpr) tagRelease(ctx context.Context, pr *github.PullRequest, currVer *semv, latestSemverTag string) error {
	var (
		vfile string
		err   error
	)
	releaseBranch := rp.cfg.releaseBranch.String()

	// Using "HEAD~" to retrieve the one previous commit before merging does not work well in cases
	// "Rebase and merge" was used. However, we don't care about "Rebase and merge" and only support
	// "Create a merge commit" and "Squash and merge."
	if rp.cfg.versionFile == nil {
		rp.c.Git("checkout", "HEAD~")
		vfile, err = detectVersionFile(".", currVer)
		if err != nil {
			return err
		}
		rp.c.Git("checkout", releaseBranch)
	} else {
		vfile = rp.cfg.versionFile.String()
	}

	var nextTag string
	if vfile != "" {
		nextVer, err := retrieveVersionFromFile(vfile, currVer.vPrefix)
		if err != nil {
			return err
		}
		nextTag = nextVer.Tag()
	} else {
		nextTag = guessNextSemver(currVer, pr.Labels).Tag()
	}
	previousTag := &latestSemverTag
	if *previousTag == "" {
		previousTag = nil
	}

	// To avoid putting pull requests created by rcpr itself in the release notes,
	// we generate release notes in advance.
	// Get the previous commitish to avoid picking up the merge of the pull
	// request made by rcpr.
	targetCommitish, _, err := rp.c.GitE("rev-parse", "HEAD~")
	if err != nil {
		return nil
	}
	releases, _, err := rp.gh.Repositories.GenerateReleaseNotes(
		ctx, rp.owner, rp.repo, &github.GenerateNotesOptions{
			TagName:         nextTag,
			PreviousTagName: previousTag,
			TargetCommitish: &targetCommitish,
		})
	if err != nil {
		return err
	}

	rp.c.Git("tag", nextTag)
	if rp.c.err != nil {
		return rp.c.err
	}
	_, _, err = rp.c.GitE("push", "--tags")
	if err != nil {
		return err
	}

	// Don't use GenerateReleaseNote flag and use pre generated one
	_, _, err = rp.gh.Repositories.CreateRelease(
		ctx, rp.owner, rp.repo, &github.RepositoryRelease{
			TagName:         &nextTag,
			TargetCommitish: &releaseBranch,
			Name:            &releases.Name,
			Body:            &releases.Body,
			// I want to make it as a draft release by default, but it is difficult to get a draft release
			// from another tool via API, and there is no tool supports it, so I will make it as a normal
			// release. In the future, there may be an option to create it as a Draft, or conversely,
			// an option not to create a release.
			// Draft: github.Bool(true),
		})
	return err
}
