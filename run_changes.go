package tagpr

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/go-github/v93/github"
)

func (tp *tagpr) mergedChangeLabels(ctx context.Context, fromCommitish, releaseBranch string) ([]string, error) {
	mergeLogArgs := tagLogArgs(tp.normalizedTagPrefix, "log", "--merges", "--first-parent", "--pretty=format:%P",
		fmt.Sprintf("%s..%s/%s", fromCommitish, tp.remoteName, releaseBranch))
	shasStr, _, err := tp.c.Git(mergeLogArgs...)
	if err != nil {
		return nil, err
	}
	mergedFeatureHeadShas := mergedFeatureHeads(shasStr)
	prShasStr, _, err := tp.c.Git("ls-remote", tp.remoteName, "refs/pull/*/head")
	if err != nil {
		return nil, err
	}

	return tp.getNextLabels(ctx, mergedFeatureHeadShas, prShasStr, fromCommitish)
}

func (tp *tagpr) planReleasePullRequest(ctx context.Context, head, releaseBranch string, currVer *semv, nextLabels []string) (*github.PullRequest, *semv, []string, error) {
	pulls, resp, err := tp.gh.PullRequests.List(ctx, tp.owner, tp.repo,
		&github.PullRequestListOptions{
			Head: head,
			Base: releaseBranch,
		})
	if err != nil {
		tp.showGHError(err, resp)
		return nil, nil, nil, err
	}

	var (
		labels    []string
		currTagPR *github.PullRequest
	)
	if len(pulls) > 0 {
		currTagPR = pulls[0]
		for _, l := range currTagPR.Labels {
			labels = append(labels, l.GetName())
		}
	}
	nextVer := currVer.GuessNext(append(labels, nextLabels...))
	var addingLabels []string

	for _, l := range nextLabels {
		if !slices.Contains(labels, l) {
			addingLabels = append(addingLabels, l)
		}
	}
	return currTagPR, nextVer, addingLabels, nil
}

func (tp *tagpr) prepareReleaseChanges(currVer, nextVer *semv, commandErr *error) ([]string, error) {
	vfiles, err := tp.getVfiles(currVer)
	if err != nil {
		return nil, err
	}

	if prog := tp.cfg.Command(); prog != "" {
		tp.runReleaseCommand(commandErr, configCommand, prog, currVer, nextVer)
	}

	if len(vfiles) > 0 && vfiles[0] != "" {
		for _, vfile := range vfiles {
			if err := bumpVersionFile(vfile, currVer, nextVer); err != nil {
				return nil, err
			}
		}
	}
	tp.c.Git("add", "-f", tp.cfg.conf) // ignore any errors

	if prog := tp.cfg.PostVersionCommand(); prog != "" {
		tp.runReleaseCommand(commandErr, configPostVersionCommand, prog, currVer, nextVer)
	}

	if err := tp.prepareReleaseYAML(); err != nil {
		return nil, err
	}
	return vfiles, nil
}

func (tp *tagpr) prepareReleaseYAML() error {
	releaseYaml := tp.cfg.ReleaseYAMLPath()
	if releaseYaml != "" && !exists(releaseYaml) {
		if err := tp.initializeReleaseYaml(releaseYaml); err != nil {
			return err
		}
	}
	if releaseYaml == "" {
		// GitHub supports both default filenames when configuration_file_path is omitted.
		// https://docs.github.com/en/rest/releases/releases#generate-release-notes-content-for-a-release
		const defaultReleaseYml = ".github/release.yml"
		const defaultReleaseYaml = ".github/release.yaml"
		if !exists(defaultReleaseYml) && !exists(defaultReleaseYaml) {
			if err := tp.initializeReleaseYaml(defaultReleaseYml); err != nil {
				return err
			}
		}
	}

	return nil
}

func (tp *tagpr) createPreparationCommit(ctx context.Context, ref *github.Reference, commitMessage string) (*github.Commit, error) {
	diffFiles, _, err := tp.c.Git("diff", "--raw", "HEAD")
	if err != nil {
		return nil, err
	}
	treeEntries, err := treeEntriesFromDiff(diffFiles)
	if err != nil {
		return nil, err
	}

	var tree *github.Tree
	var resp *github.Response
	if len(treeEntries) > 0 {
		tree, resp, err = tp.gh.Git.CreateTree(ctx, tp.owner, tp.repo, *ref.Object.SHA, treeEntries)
		if err != nil {
			tp.showGHError(err, resp)
			return nil, err
		}
	}

	parent, resp, err := tp.gh.Repositories.GetCommit(ctx, tp.owner, tp.repo, *ref.Object.SHA, nil)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}
	parent.Commit.SHA = parent.SHA

	commit := github.Commit{
		Message: new(commitMessage),
		Tree:    parent.Commit.Tree,
		Parents: []*github.Commit{parent.Commit},
	}
	if tree != nil {
		commit.Tree = tree
	}
	newCommit, resp, err := tp.gh.Git.CreateCommit(ctx, tp.owner, tp.repo, commit, nil)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}

	return newCommit, nil
}

func (tp *tagpr) reevaluateReleaseVersion(currVer, nextVer *semv, vfiles []string) (*semv, error) {
	// Reread the configuration file (.tagpr) as it may have been rewritten during the cherry-pick process.
	tp.cfg.Reload()
	tp.normalizedTagPrefix = normalizeTagPrefix(tp.cfg.TagPrefix())
	if tp.cfg.VersionFile() != "" && tp.cfg.VersionFile() != "-" {
		vfiles = splitVersionFiles(tp.cfg.VersionFile())
	}
	if len(vfiles) > 0 && vfiles[0] != "" {
		if tp.cfg.CalendarVersioning() {
			for _, vfile := range vfiles {
				if err := bumpVersionFile(vfile, currVer, nextVer); err != nil {
					return nil, err
				}
			}
		} else {
			nVer, _ := retrieveVersionFromFile(vfiles[0], nextVer)
			if nVer != nil && nVer.Naked() != nextVer.Naked() {
				nextVer = nVer
			}
		}
	}

	return nextVer, nil
}
