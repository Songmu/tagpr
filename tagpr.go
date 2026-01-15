package tagpr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/Songmu/gh2changelog"
	"github.com/Songmu/gitconfig"
	"github.com/google/go-github/v74/github"
)

const (
	defaultReleaseBranch = "main"
	autoCommitMessage    = "prepare for the next release"
	autoChangelogMessage = "update CHANGELOG.md"
	autoLabelName        = "tagpr"
	branchPrefix         = "tagpr-from-"
)

type tagpr struct {
	c                       *commander
	gh                      *github.Client
	cfg                     *config
	gitPath                 string
	remoteName, owner, repo string
	out                     io.Writer
	normalizedTagPrefix     string
}

func newTagPR(ctx context.Context, c *commander) (*tagpr, error) {
	tp := &tagpr{c: c, gitPath: c.gitPath, out: c.outStream}

	var err error
	tp.remoteName, err = tp.detectRemote()
	if err != nil {
		return nil, err
	}
	remoteURL, _, err := tp.c.Git("config", "remote."+tp.remoteName+".url")
	if err != nil {
		return nil, err
	}
	u, err := parseGitURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote")
	}
	m := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(m) < 2 {
		return nil, fmt.Errorf("failed to detect owner and repo from remote URL")
	}
	tp.owner = m[0]
	repo := m[1]
	if u.Scheme == "ssh" || u.Scheme == "git" {
		repo = strings.TrimSuffix(repo, ".git")
	}
	tp.repo = repo

	token, err := gitconfig.GitHubToken(u.Hostname())
	if err != nil {
		return nil, err
	}
	cli, err := ghClient(ctx, token, u.Host)
	if err != nil {
		return nil, err
	}
	tp.gh = cli

	// pass u.Host instead of host because u.Host includes port number if exists.
	tp.c.SetToken(token, u.Host)

	isShallow, _, err := tp.c.Git("rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, err
	}
	if isShallow == "true" {
		if _, _, err := tp.c.Git("fetch", "--unshallow"); err != nil {
			return nil, err
		}
	}
	tp.cfg, err = newConfig(tp.gitPath)
	if err != nil {
		return nil, err
	}
	tp.normalizedTagPrefix = normalizeTagPrefix(tp.cfg.TagPrefix())
	return tp, nil
}

func (tp *tagpr) Run(ctx context.Context) error {
	commitMessage := tp.cfg.CommitPrefix() + " " + autoCommitMessage
	changelogMessage := tp.cfg.CommitPrefix() + " " + autoChangelogMessage

	latestSemverTag := tp.latestSemverTag()
	tp.setOutput("base_tag", latestSemverTag)
	currVerStr := latestSemverTag
	fromCommitish := "refs/tags/" + currVerStr
	if currVerStr == "" {
		var err error
		fromCommitish, _, err = tp.c.Git("rev-list", "--max-parents=0", "HEAD")
		if err != nil {
			return err
		}
		currVerStr = "v0.0.0"
	} else {
		// Strip prefix for newSemver (fromCommitish already has full tag name)
		currVerStr = strings.TrimPrefix(currVerStr, tp.normalizedTagPrefix)
	}
	currVer, err := newSemver(currVerStr)
	if err != nil {
		return err
	}

	if tp.cfg.vPrefix == nil {
		if err := tp.cfg.SetVPrefix(currVer.vPrefix); err != nil {
			return err
		}
	} else {
		currVer.vPrefix = *tp.cfg.vPrefix
	}

	releaseBranch := tp.cfg.ReleaseBranch()
	if releaseBranch == "" {
		releaseBranch, _ = tp.defaultBranch()
		if releaseBranch == "" {
			releaseBranch = defaultReleaseBranch
		}
		if err := tp.cfg.SetReleaseBranch(releaseBranch); err != nil {
			return err
		}
	}

	branch, _, err := tp.c.Git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to git symbolic-ref: %w", err)
	}
	if branch != releaseBranch {
		return fmt.Errorf("you are not on release branch %q, current branch is %q",
			releaseBranch, branch)
	}

	// If the latest commit is a merge commit of the pull request by tagpr,
	// tag the semver to the commit and create a release and exit.
	if pr, err := tp.latestPullRequest(ctx); err != nil || isTagPR(pr) {
		if err != nil {
			return err
		}
		if err := tp.tagRelease(ctx, pr, currVer, latestSemverTag); err != nil {
			return err
		}
		b, _ := json.Marshal(pr)
		tp.setOutput("pull_request", string(b))
		return nil
	}
	mergeLogArgs := []string{"log", "--merges", "--pretty=format:%P",
		fmt.Sprintf("%s..%s/%s", fromCommitish, tp.remoteName, releaseBranch)}
	if tp.normalizedTagPrefix != "" {
		mergeLogArgs = append(mergeLogArgs, "--", strings.TrimSuffix(tp.normalizedTagPrefix, "/"))
	}
	shasStr, _, err := tp.c.Git(mergeLogArgs...)
	if err != nil {
		return err
	}
	var mergedFeatureHeadShas []string
	for line := range strings.SplitSeq(shasStr, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		mergedFeatureHeadShas = append(mergedFeatureHeadShas, fields[1])
	}
	prShasStr, _, err := tp.c.Git("ls-remote", tp.remoteName, "refs/pull/*/head")
	if err != nil {
		return err
	}

	nextLabels, err := tp.getNextLabels(ctx, mergedFeatureHeadShas, prShasStr, fromCommitish)
	if err != nil {
		return err
	}

	// Get the latest commit of the release branch
	ref, resp, err := tp.gh.Git.GetRef(ctx, tp.owner, tp.repo, "refs/heads/"+releaseBranch)
	if err != nil {
		showGHError(err, resp)
		return err
	}

	rcBranch := fmt.Sprintf("%s%s%s", branchPrefix, branchSafePrefix(tp.normalizedTagPrefix), currVer.Tag())
	head := fmt.Sprintf("%s:%s", tp.owner, rcBranch)
	pulls, resp, err := tp.gh.PullRequests.List(ctx, tp.owner, tp.repo,
		&github.PullRequestListOptions{
			Head: head,
			Base: releaseBranch,
		})
	if err != nil {
		showGHError(err, resp)
		return err
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

	var vfiles []string
	if vf := tp.cfg.VersionFile(); vf != "" && vf != "-" {
		vfiles = strings.Split(vf, ",")
		for i, v := range vfiles {
			vfiles[i] = strings.TrimSpace(v)
		}
	} else if tp.cfg.versionFile == nil {
		vfile, err := detectVersionFile(".", currVer)
		if err != nil {
			return err
		}
		if err := tp.cfg.SetVersionFile(vfile); err != nil {
			return err
		}
		vfiles = []string{vfile}
	}

	if prog := tp.cfg.Command(); prog != "" {
		tp.Exec(prog, currVer, nextVer)
	}

	if len(vfiles) > 0 && vfiles[0] != "" {
		for _, vfile := range vfiles {
			if err := bumpVersionFile(vfile, currVer, nextVer); err != nil {
				return err
			}
		}
	}
	tp.c.Git("add", "-f", tp.cfg.conf) // ignore any errors

	if prog := tp.cfg.PostVersionCommand(); prog != "" {
		tp.Exec(prog, currVer, nextVer)
	}

	const releaseYml = ".github/release.yml"
	const releaseYaml = ".github/release.yaml"
	// TODO: It would be nice to be able to add an exclude setting even if release.yml already exists.
	if !exists(releaseYml) && !exists(releaseYaml) {
		if err := os.MkdirAll(filepath.Dir(releaseYml), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(releaseYml, []byte(`changelog:
  exclude:
    labels:
      - tagpr
`), 0644); err != nil {
			return err
		}
		tp.c.Git("add", "-f", releaseYml)
	}

	// Detect modified files and create a new tree object
	diffFiles, _, err := tp.c.Git("diff", "--raw", "HEAD")
	if err != nil {
		return err
	}
	var treeEntries []*github.TreeEntry
	for line := range strings.SplitSeq(diffFiles, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, ":") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}

		newMode, status, filePath := parts[1], parts[4], parts[5]
		switch status {
		case "A", "M": // Created or modified files
			contentBytes, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			treeEntries = append(treeEntries, &github.TreeEntry{
				Path:    github.Ptr(filePath),
				Type:    github.Ptr("blob"),
				Content: github.Ptr(string(contentBytes)),
				Mode:    github.Ptr(newMode),
			})
		case "D": // Deleted files
			treeEntries = append(treeEntries, &github.TreeEntry{
				SHA:  nil,
				Path: github.Ptr(filePath),
				Type: github.Ptr("blob"),
				Mode: github.Ptr("100644"),
			})
		}
	}

	var tree *github.Tree
	if len(treeEntries) > 0 {
		// Create a new tree object if there are changes
		tree, resp, err = tp.gh.Git.CreateTree(ctx, tp.owner, tp.repo, *ref.Object.SHA, treeEntries)
		if err != nil {
			showGHError(err, resp)
			return err
		}
	}

	// Get the parent commit to attach the commit to.
	parent, resp, err := tp.gh.Repositories.GetCommit(ctx, tp.owner, tp.repo, *ref.Object.SHA, nil)
	if err != nil {
		showGHError(err, resp)
		return err
	}
	parent.Commit.SHA = parent.SHA

	// Create a new commit
	commit := &github.Commit{
		Message: github.Ptr(commitMessage),
		Tree:    parent.Commit.Tree,
		Parents: []*github.Commit{parent.Commit},
	}
	if tree != nil {
		commit.Tree = tree
	}
	newCommit, resp, err := tp.gh.Git.CreateCommit(ctx, tp.owner, tp.repo, commit, nil)
	if err != nil {
		showGHError(err, resp)
		return err
	}

	// cherry-pick if the remote branch is exists and changed
	// XXX: Do I need to apply merge commits too?
	//     (We omitted merge commits for now, because if we cherry-pick them, we need to add options like "-m 1".
	cherryLogArgs := []string{"log", "--no-merges", "--pretty=format:%h %s",
		fmt.Sprintf("%s..%s/%s", releaseBranch, tp.remoteName, rcBranch)}
	if tp.normalizedTagPrefix != "" {
		cherryLogArgs = append(cherryLogArgs, "--", strings.TrimSuffix(tp.normalizedTagPrefix, "/"))
	}
	out, _, err := tp.c.Git(cherryLogArgs...)
	if err == nil {
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
		if len(cherryPicks) > 0 {
			// Specify a commitish one by one for cherry-pick instead of multiple commitish,
			// and apply it as much as possible.

			// Delete temporary reference if it exists
			resp, err := tp.gh.Git.DeleteRef(ctx, tp.owner, tp.repo, "refs/heads/tagpr-temp")
			if err != nil && resp.StatusCode != 422 {
				showGHError(err, resp)
				return err
			}
			// Create a temporary reference
			tempRef := &github.Reference{
				Ref:    github.Ptr("refs/heads/tagpr-temp"),
				Object: &github.GitObject{SHA: newCommit.SHA},
			}
			tempRef, resp, err = tp.gh.Git.CreateRef(ctx, tp.owner, tp.repo, tempRef)
			if err != nil {
				showGHError(err, resp)
				return err
			}

			for i := len(cherryPicks) - 1; i >= 0; i-- {
				commitish := cherryPicks[i]

				// Get cherry-pick commit
				cherryPickCommit, resp, err := tp.gh.Repositories.GetCommit(
					ctx, tp.owner, tp.repo, commitish, nil)
				if err != nil {
					showGHError(err, resp)
					return err
				}

				// Create a new commit
				commit := &github.Commit{
					Message: github.Ptr("cherry-pick: " + commitish),
					Tree:    newCommit.Tree,
					Parents: cherryPickCommit.Parents,
				}
				tempCommit, resp, err := tp.gh.Git.CreateCommit(ctx, tp.owner, tp.repo, commit, nil)
				if err != nil {
					showGHError(err, resp)
					return err
				}

				// Update temporary reference
				tempRef.Object.SHA = tempCommit.SHA
				_, resp, err = tp.gh.Git.UpdateRef(ctx, tp.owner, tp.repo, tempRef, true)
				if err != nil {
					showGHError(err, resp)
					return err
				}

				// Merge
				mergeRequest := &github.RepositoryMergeRequest{
					Base: github.Ptr("tagpr-temp"),
					Head: github.Ptr(commitish),
				}
				mergeCommit, resp, err := tp.gh.Repositories.Merge(
					ctx, tp.owner, tp.repo, mergeRequest)
				if err != nil {
					// conflict, etc. / Need error handling in case of non-conflict error?
					if resp.StatusCode == 409 {
						continue
					}
					showGHError(err, resp)
					return err
				}

				// Create a new commit
				// The Author is not set because setting the same Author as the original commit makes it
				// difficult to create a Verified Commit.
				commit = &github.Commit{
					Message: cherryPickCommit.Commit.Message,
					Tree:    mergeCommit.Commit.Tree,
					Parents: []*github.Commit{newCommit},
				}
				newCommit, resp, err = tp.gh.Git.CreateCommit(ctx, tp.owner, tp.repo, commit, nil)
				if err != nil {
					showGHError(err, resp)
					return err
				}

				// Update temporary reference
				tempRef.Object.SHA = newCommit.SHA
				_, resp, err = tp.gh.Git.UpdateRef(ctx, tp.owner, tp.repo, tempRef, true)
				if err != nil {
					showGHError(err, resp)
					return err
				}
			}

			// Checkout the temporary reference (Files like .tagpr used in subsequent processes may have
			// been rewritten during the cherry-pick process)
			if _, _, err := tp.c.Git("fetch"); err != nil {
				return err
			}
			if _, _, err := tp.c.Git("reset", "--hard"); err != nil {
				return err
			}
			if _, _, err := tp.c.Git("checkout", "tagpr-temp"); err != nil {
				return err
			}

			// Delete temporary reference
			resp, err = tp.gh.Git.DeleteRef(ctx, tp.owner, tp.repo, "refs/heads/tagpr-temp")
			if err != nil {
				showGHError(err, resp)
				return err
			}
		}
	}

	// Reread the configuration file (.tagpr) as it may have been rewritten during the cherry-pick process.
	tp.cfg.Reload()
	tp.normalizedTagPrefix = normalizeTagPrefix(tp.cfg.TagPrefix())
	if tp.cfg.VersionFile() != "" && tp.cfg.VersionFile() != "-" {
		vfiles = strings.Split(tp.cfg.VersionFile(), ",")
		for i, v := range vfiles {
			vfiles[i] = strings.TrimSpace(v)
		}
	}
	if len(vfiles) > 0 && vfiles[0] != "" {
		nVer, _ := retrieveVersionFromFile(vfiles[0], nextVer.vPrefix)
		if nVer != nil && nVer.Naked() != nextVer.Naked() {
			nextVer = nVer
		}
	}

	gch, err := gh2changelog.New(ctx,
		gh2changelog.GitPath(tp.gitPath),
		gh2changelog.SetOutputs(tp.c.outStream, tp.c.errStream),
		gh2changelog.GitHubClient(tp.gh),
		gh2changelog.TagPrefix(tp.normalizedTagPrefix),
	)
	if err != nil {
		return err
	}

	draftNextTag := fullTag(tp.normalizedTagPrefix, nextVer.Tag())
	changelog, orig, err := gch.Draft(ctx, draftNextTag, releaseBranch, time.Now())
	if err != nil {
		return err
	}

	if tp.cfg.changelog == nil || *tp.cfg.changelog {
		changelogMd := "CHANGELOG.md"
		if !exists(changelogMd) {
			logs, _, err := gch.Changelogs(ctx, 20)
			if err != nil {
				return err
			}
			changelog = strings.Join(
				append([]string{changelog}, logs...), "\n")
		}
		if _, err := gch.Update(changelog, 0); err != nil {
			return err
		}

		// Create a new tree object for CHANGELOG.md
		treeEntries = nil
		contentBytes, err := os.ReadFile(changelogMd)
		if err != nil {
			return err
		}
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path:    github.Ptr(changelogMd),
			Type:    github.Ptr("blob"),
			Content: github.Ptr(string(contentBytes)),
			Mode:    github.Ptr("100644"),
		})
		tree, resp, err = tp.gh.Git.CreateTree(ctx, tp.owner, tp.repo, *newCommit.SHA, treeEntries)
		if err != nil {
			showGHError(err, resp)
			return err
		}
		// Create a new commit
		commit = &github.Commit{
			Message: github.Ptr(changelogMessage),
			Tree:    tree,
			Parents: []*github.Commit{newCommit},
		}
		newCommit, resp, err = tp.gh.Git.CreateCommit(ctx, tp.owner, tp.repo, commit, nil)
		if err != nil {
			showGHError(err, resp)
			return err
		}
	}

	// Create or Get remote rcBranch reference
	rcBranchRef, resp, err := tp.gh.Git.GetRef(ctx, tp.owner, tp.repo, "refs/heads/"+rcBranch)
	if err != nil {
		if resp.StatusCode != 404 {
			showGHError(err, resp)
			return err
		}
		newRef := &github.Reference{
			Ref:    github.Ptr("refs/heads/" + rcBranch),
			Object: ref.Object,
		}
		rcBranchRef, resp, err = tp.gh.Git.CreateRef(ctx, tp.owner, tp.repo, newRef)
		if err != nil {
			showGHError(err, resp)
			return err
		}
	}
	// Force update the rcBranch reference
	rcBranchRef.Object.SHA = newCommit.SHA
	_, resp, err = tp.gh.Git.UpdateRef(ctx, tp.owner, tp.repo, rcBranchRef, true)
	if err != nil {
		showGHError(err, resp)
		return err
	}

	var tmpl *template.Template
	if t := tp.cfg.Template(); t != "" {
		tmpTmpl, err := template.ParseFiles(t)
		if err == nil {
			tmpl = tmpTmpl
		} else {
			log.Printf("parse configured template failed: %s\n", err)
		}
	} else if t := tp.cfg.TemplateText(); t != "" {
		tmpTmplTxt, err := template.New("templateText").Parse(t)
		if err == nil {
			tmpl = tmpTmplTxt
		} else {
			log.Printf("parse configured template failed: %s\n", err)
		}
	}

	host := "github.com"
	if tp.gh.BaseURL != nil {
		host = strings.TrimPrefix(tp.gh.BaseURL.Host, "api.")
	}
	currTag := fullTag(tp.normalizedTagPrefix, currVer.Tag())
	nextTag := fullTag(tp.normalizedTagPrefix, nextVer.Tag())
	orig = replaceCompareLink(orig, host, tp.owner, tp.repo, currTag, nextTag, rcBranch)
	pt := newPRTmpl(tmpl)
	prText, err := pt.Render(&tmplArg{
		NextVersion: nextVer.Tag(),
		Branch:      rcBranch,
		Changelog:   orig,
		TagPrefix:   strings.TrimSuffix(tp.normalizedTagPrefix, "/"),
	})
	if err != nil {
		return err
	}

	stuffs := strings.SplitN(strings.TrimSpace(prText), "\n", 2)
	title := stuffs[0]
	var body string
	if len(stuffs) > 1 {
		body = strings.TrimSpace(stuffs[1])
	}
	if currTagPR == nil {
		pr, resp, err := tp.gh.PullRequests.Create(ctx, tp.owner, tp.repo, &github.NewPullRequest{
			Title: github.Ptr(title),
			Body:  github.Ptr(body),
			Base:  &releaseBranch,
			Head:  github.Ptr(head),
		})
		if err != nil {
			showGHError(err, resp)
			return err
		}
		addingLabels = append(addingLabels, autoLabelName)
		_, resp, err = tp.gh.Issues.AddLabelsToIssue(
			ctx, tp.owner, tp.repo, *pr.Number, addingLabels)
		if err != nil {
			showGHError(err, resp)
			return err
		}
		tmpPr, resp, err := tp.gh.PullRequests.Get(ctx, tp.owner, tp.repo, *pr.Number)
		if err == nil {
			pr = tmpPr
		} else {
			showGHError(err, resp)
		}
		b, _ := json.Marshal(pr)
		tp.setOutput("pull_request", string(b))
		return nil
	}
	currTagPR.Title = github.Ptr(title)
	currTagPR.Body = github.Ptr(mergeBody(*currTagPR.Body, body))
	pr, resp, err := tp.gh.PullRequests.Edit(ctx, tp.owner, tp.repo, *currTagPR.Number, currTagPR)
	if err != nil {
		showGHError(err, resp)
		return err
	}
	if len(addingLabels) > 0 {
		_, resp, err := tp.gh.Issues.AddLabelsToIssue(
			ctx, tp.owner, tp.repo, *currTagPR.Number, addingLabels)
		if err != nil {
			showGHError(err, resp)
			return err
		}
		tmpPr, resp, err := tp.gh.PullRequests.Get(ctx, tp.owner, tp.repo, *pr.Number)
		if err == nil {
			pr = tmpPr
		} else {
			showGHError(err, resp)
		}
	}
	b, _ := json.Marshal(pr)
	tp.setOutput("pull_request", string(b))
	return nil
}
