package tagpr

import (
	"context"

	"github.com/google/go-github/v45/github"
)

func (tp *tagpr) latestPullRequest(ctx context.Context) (*github.PullRequest, error) {
	// tag and exit if the HEAD is the merged tagpr
	commitish, _, err := tp.c.GitE("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	pulls, _, err := tp.gh.PullRequests.ListPullRequestsWithCommit(
		ctx, tp.owner, tp.repo, commitish, nil)
	if err != nil {
		return nil, err
	}
	if len(pulls) == 0 {
		return nil, nil
	}
	return pulls[0], nil
}

func (tp *tagpr) tagRelease(ctx context.Context, pr *github.PullRequest, currVer *semv, latestSemverTag string) error {
	var (
		vfile string
		err   error
	)
	releaseBranch := tp.cfg.releaseBranch.String()

	// Using "HEAD~" to retrieve the one previous commit before merging does not work well in cases
	// "Rebase and merge" was used. However, we don't care about "Rebase and merge" and only support
	// "Create a merge commit" and "Squash and merge."
	if tp.cfg.versionFile == nil {
		tp.c.Git("checkout", "HEAD~")
		vfile, err = detectVersionFile(".", currVer)
		if err != nil {
			return err
		}
		tp.c.Git("checkout", releaseBranch)
	} else {
		vfile = tp.cfg.versionFile.String()
	}

	var nextTag string
	if vfile != "" {
		nextVer, err := retrieveVersionFromFile(vfile, currVer.vPrefix)
		if err != nil {
			return err
		}
		nextTag = nextVer.Tag()
	} else {
		nextTag = currVer.GuessNext(pr.Labels).Tag()
	}
	previousTag := &latestSemverTag
	if *previousTag == "" {
		previousTag = nil
	}

	// To avoid putting pull requests created by tagpr itself in the release notes,
	// we generate release notes in advance.
	// Get the previous commitish to avoid picking up the merge of the pull
	// request made by tagpr.
	targetCommitish, _, err := tp.c.GitE("rev-parse", "HEAD~")
	if err != nil {
		return nil
	}
	releases, _, err := tp.gh.Repositories.GenerateReleaseNotes(
		ctx, tp.owner, tp.repo, &github.GenerateNotesOptions{
			TagName:         nextTag,
			PreviousTagName: previousTag,
			TargetCommitish: &targetCommitish,
		})
	if err != nil {
		return err
	}

	tp.c.Git("tag", nextTag)
	if tp.c.err != nil {
		return tp.c.err
	}
	_, _, err = tp.c.GitE("push", "--tags")
	if err != nil {
		return err
	}

	// Don't use GenerateReleaseNote flag and use pre generated one
	_, _, err = tp.gh.Repositories.CreateRelease(
		ctx, tp.owner, tp.repo, &github.RepositoryRelease{
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
