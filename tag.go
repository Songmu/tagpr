package tagpr

import (
	"context"
	"strings"

	"github.com/google/go-github/v49/github"
)

func (tp *tagpr) latestPullRequest(ctx context.Context) (*github.PullRequest, error) {
	// tag and exit if the HEAD is the merged tagpr
	commitish, _, err := tp.c.Git("rev-parse", "HEAD")
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
	releaseBranch := tp.cfg.ReleaseBranch()

	// Using "HEAD~" to retrieve the one previous commit before merging does not work well in cases
	// "Rebase and merge" was used. However, we don't care about "Rebase and merge" and only support
	// "Create a merge commit" and "Squash and merge."
	if tp.cfg.VersionFile() == "" {
		if _, _, err := tp.c.Git("checkout", "HEAD~"); err != nil {
			return err
		}
		vfile, err = detectVersionFile(".", currVer)
		if err != nil {
			return err
		}
		if _, _, err := tp.c.Git("checkout", releaseBranch); err != nil {
			return err
		}
	} else {
		vfiles := strings.Split(tp.cfg.VersionFile(), ",")
		vfile = strings.TrimSpace(vfiles[0])
	}

	var nextTag string
	if vfile != "" {
		nextVer, err := retrieveVersionFromFile(vfile, currVer.vPrefix)
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
	previousTag := &latestSemverTag
	if *previousTag == "" {
		previousTag = nil
	}

	// To avoid putting pull requests created by tagpr itself in the release notes,
	// we generate release notes in advance.
	// Get the previous commitish to avoid picking up the merge of the pull
	// request made by tagpr.
	targetCommitish, _, err := tp.c.Git("rev-parse", "HEAD~")
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

	if _, _, err := tp.c.Git("tag", nextTag); err != nil {
		return err
	}
	_, _, err = tp.c.Git("push", "--tags")
	if err != nil {
		return err
	}
	tp.setOutput("tag", nextTag)

	if !tp.cfg.Release() {
		return nil
	}
	// Don't use GenerateReleaseNote flag and use pre generated one
	_, _, err = tp.gh.Repositories.CreateRelease(
		ctx, tp.owner, tp.repo, &github.RepositoryRelease{
			TagName:         &nextTag,
			TargetCommitish: &releaseBranch,
			Name:            &releases.Name,
			Body:            &releases.Body,
			Draft:           github.Bool(tp.cfg.ReleaseDraft()),
		})
	return err
}
