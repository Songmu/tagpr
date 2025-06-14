package tagpr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Songmu/gh2changelog"
	"github.com/Songmu/gitsemvers"
	"github.com/google/go-github/v66/github"
)

const (
	gitUser              = "github-actions[bot]"
	gitEmail             = "github-actions[bot]@users.noreply.github.com"
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
}

func (tp *tagpr) latestSemverTag() string {
	vers := (&gitsemvers.Semvers{GitPath: tp.gitPath}).VersionStrings()
	if tp.cfg.vPrefix != nil {
		for _, v := range vers {
			if strings.HasPrefix(v, "v") == *tp.cfg.vPrefix {
				return v
			}
		}
	} else {
		// When vPrefix is not defined (i.e. first time tagpr setup), just return the first value.
		if len(vers) > 0 {
			return vers[0]
		}
	}
	return ""
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

	cli, err := ghClient(ctx, "", u.Hostname())
	if err != nil {
		return nil, err
	}
	tp.gh = cli

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
	return tp, nil
}

func isTagPR(pr *github.PullRequest) bool {
	if pr == nil || pr.Head == nil || pr.Head.Ref == nil || !strings.HasPrefix(*pr.Head.Ref, branchPrefix) {
		return false
	}
	for _, label := range pr.Labels {
		if label.GetName() == autoLabelName {
			return true
		}
	}
	return false
}

func (tp *tagpr) Run(ctx context.Context) error {
	commitMessage := tp.cfg.CommitPrefix() + " " + autoCommitMessage
	changelogMessage := tp.cfg.CommitPrefix() + " " + autoChangelogMessage

	latestSemverTag := tp.latestSemverTag()
	currVerStr := latestSemverTag
	fromCommitish := "refs/tags/" + currVerStr
	if currVerStr == "" {
		var err error
		fromCommitish, _, err = tp.c.Git("rev-list", "--max-parents=0", "HEAD")
		if err != nil {
			return err
		}
		currVerStr = "v0.0.0"
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
	shasStr, _, err := tp.c.Git("log", "--merges", "--pretty=format:%P",
		fmt.Sprintf("%s..%s/%s", fromCommitish, tp.remoteName, releaseBranch))
	if err != nil {
		return err
	}
	var mergedFeatureHeadShas []string
	for _, line := range strings.Split(shasStr, "\n") {
		stuff := strings.Fields(line)
		if len(stuff) < 2 {
			continue
		}
		mergedFeatureHeadShas = append(mergedFeatureHeadShas, stuff[1])
	}
	prShasStr, _, err := tp.c.Git("ls-remote", tp.remoteName, "refs/pull/*/head")
	if err != nil {
		return err
	}
	var prIssues []*github.Issue
	for _, line := range strings.Split(prShasStr, "\n") {
		stuff := strings.Fields(line)
		if len(stuff) != 2 {
			continue
		}
		sha, ref := stuff[0], stuff[1]
		for _, mergedSha := range mergedFeatureHeadShas {
			if strings.HasPrefix(sha, mergedSha) {
				prNumStr := strings.Trim(ref, "head/rfspul")
				prNum, err := strconv.Atoi(prNumStr)
				if err != nil {
					continue
				}
				issue, resp, err := tp.gh.Issues.Get(ctx, tp.owner, tp.repo, prNum)
				if err != nil {
					showGHError(err, resp)
					return err
				}
				prIssues = append(prIssues, issue)
			}
		}
	}
	// When "--abbrev" is specified, the length of the each line of the stdout isn't fixed.
	// It is just a minimum length, and if the commit cannot be uniquely identified with
	// that length, a longer commit hash will be displayed.
	// We specify this option to minimize the length of the query string, but we use
	// "--abbrev=7" because the SHA syntax of the search API requires a string of at
	// least 7 characters.
	// ref. https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests#search-by-commit-sha
	// This is done because there is a length limit on the API query string, and we want
	// to create a string with the minimum possible length.
	shasStr, _, err = tp.c.Git("log", "--pretty=format:%h", "--abbrev=7", "--no-merges", "--first-parent",
		fmt.Sprintf("%s..%s/%s", fromCommitish, tp.remoteName, releaseBranch))
	if err != nil {
		return err
	}
	queryBase := fmt.Sprintf("repo:%s/%s is:pr is:closed", tp.owner, tp.repo)
	for _, query := range buildChunkSearchIssuesQuery(queryBase, shasStr) {
		tmpIssues, err := tp.searchIssues(ctx, query)
		if err != nil {
			return err
		}
		prIssues = append(prIssues, tmpIssues...)
	}

	nextLabels := tp.generatenNextLabels(prIssues)

	// Get the latest commit of the release branch
	ref, resp, err := tp.gh.Git.GetRef(ctx, tp.owner, tp.repo, "refs/heads/"+releaseBranch)
	if err != nil {
		showGHError(err, resp)
		return err
	}

	rcBranch := fmt.Sprintf("%s%s", branchPrefix, currVer.Tag())
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
OUT:
	for _, l := range nextLabels {
		for _, l2 := range labels {
			if l == l2 {
				continue OUT
			}
		}
		addingLabels = append(addingLabels, l)
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
		var progArgs []string
		if strings.ContainsAny(prog, " \n") {
			progArgs = []string{"-c", prog}
			prog = "sh"
		}
		tp.c.Cmd(prog, progArgs, map[string]string{
			"TAGPR_CURRENT_VERSION": currVer.Tag(),
			"TAGPR_NEXT_VERSION":    nextVer.Tag(),
		})
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
		var progArgs []string
		if strings.ContainsAny(prog, " \n") {
			progArgs = []string{"-c", prog}
			prog = "sh"
		}
		tp.c.Cmd(prog, progArgs, map[string]string{
			"TAGPR_CURRENT_VERSION": currVer.Tag(),
			"TAGPR_NEXT_VERSION":    nextVer.Tag(),
		})
	}

	const releaseYml = ".github/release.yml"
	// TODO: It would be nice to be able to add an exclude setting even if release.yml already exists.
	if !exists(releaseYml) {
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
	diffFiles, _, err := tp.c.Git("diff", "--name-status", "HEAD")
	if err != nil {
		return err
	}
	var treeEntries []*github.TreeEntry
	for _, line := range strings.Split(diffFiles, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status, filePath := parts[0], parts[1]
		switch status {
		case "A", "M": // Created or modified files
			contentBytes, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			treeEntries = append(treeEntries, &github.TreeEntry{
				Path: github.String(filePath), Type: github.String("blob"), Content: github.String(string(contentBytes)), Mode: github.String("100644"),
			})
		case "D": // Deleted files
			treeEntries = append(treeEntries, &github.TreeEntry{
				SHA: nil, Path: github.String(filePath), Type: github.String("blob"), Mode: github.String("100644"),
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
		Message: github.String(commitMessage),
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
	out, _, err := tp.c.Git("log", "--no-merges", "--pretty=format:%h %s",
		fmt.Sprintf("%s..%s/%s", releaseBranch, tp.remoteName, rcBranch))
	if err == nil {
		var cherryPicks []string
		for _, line := range strings.Split(out, "\n") {
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
				Ref:    github.String("refs/heads/tagpr-temp"),
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
				cherryPickCommit, resp, err := tp.gh.Repositories.GetCommit(ctx, tp.owner, tp.repo, commitish, nil)
				if err != nil {
					showGHError(err, resp)
					return err
				}

				// Create a new commit
				commit := &github.Commit{
					Message: github.String("cherry-pick: " + commitish),
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
					Base: github.String("tagpr-temp"),
					Head: github.String(commitish),
				}
				mergeCommit, resp, err := tp.gh.Repositories.Merge(ctx, tp.owner, tp.repo, mergeRequest)
				if err != nil {
					// conflict, etc. / Need error handling in case of non-conflict error?
					if resp.StatusCode == 409 {
						continue
					}
					showGHError(err, resp)
					return err
				}

				// Create a new commit
				// The Author is not set because setting the same Author as the original commit makes it difficult to create a Verified Commit.
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

			// Checkout the temporary reference (Files like .tagpr used in subsequent processes may have been rewritten during the cherry-pick process)
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
	)
	if err != nil {
		return err
	}

	changelog, orig, err := gch.Draft(ctx, nextVer.Tag(), time.Now())
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
			Path: github.String(changelogMd), Type: github.String("blob"), Content: github.String(string(contentBytes)), Mode: github.String("100644"),
		})
		tree, resp, err = tp.gh.Git.CreateTree(ctx, tp.owner, tp.repo, *newCommit.SHA, treeEntries)
		if err != nil {
			showGHError(err, resp)
			return err
		}
		// Create a new commit
		commit = &github.Commit{
			Message: github.String(changelogMessage),
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
			Ref:    github.String("refs/heads/" + rcBranch),
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
	pt := newPRTmpl(tmpl)
	prText, err := pt.Render(&tmplArg{
		NextVersion: nextVer.Tag(),
		Branch:      rcBranch,
		Changelog:   orig,
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
			Title: github.String(title),
			Body:  github.String(body),
			Base:  &releaseBranch,
			Head:  github.String(head),
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
	currTagPR.Title = github.String(title)
	currTagPR.Body = github.String(mergeBody(*currTagPR.Body, body))
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

var (
	hasSchemeReg  = regexp.MustCompile("^[^:]+://")
	scpLikeURLReg = regexp.MustCompile("^([^@]+@)?([^:]+):(/?.+)$")
)

func parseGitURL(u string) (*url.URL, error) {
	if !hasSchemeReg.MatchString(u) {
		if m := scpLikeURLReg.FindStringSubmatch(u); len(m) == 4 {
			u = fmt.Sprintf("ssh://%s%s/%s", m[1], m[2], strings.TrimPrefix(m[3], "/"))
		}
	}
	return url.Parse(u)
}

func mergeBody(now, update string) string {
	// TODO: If there are check boxes, respect what is checked, etc.
	return update
}

var headBranchReg = regexp.MustCompile(`(?m)^\s*HEAD branch: (.*)$`)

func (tp *tagpr) defaultBranch() (string, error) {
	// `git symbolic-ref refs/remotes/origin/HEAD` sometimes doesn't work
	// So use `git remote show origin` for detecting default branch
	show, _, err := tp.c.Git("remote", "show", tp.remoteName)
	if err != nil {
		return "", fmt.Errorf("failed to detect default branch: %w", err)
	}
	m := headBranchReg.FindStringSubmatch(show)
	if len(m) < 2 {
		return "", fmt.Errorf("failed to detect default branch from remote: %s", tp.remoteName)
	}
	return m[1], nil
}

func (tp *tagpr) detectRemote() (string, error) {
	remotesStr, _, err := tp.c.Git("remote")
	if err != nil {
		return "", fmt.Errorf("failed to detect remote: %s", err)
	}
	remotes := strings.Fields(remotesStr)
	if len(remotes) < 1 {
		return "", errors.New("failed to detect remote")
	}
	for _, r := range remotes {
		if r == "origin" {
			return r, nil
		}
	}
	// the last output is the first added remote
	return remotes[len(remotes)-1], nil
}

func (tp *tagpr) searchIssues(ctx context.Context, query string) ([]*github.Issue, error) {
	// Fortunately, we don't need to take care of the page count in response, because
	// the default value of per_page is 30 and we can't specify more than 30 commits due to
	// the length limit specification of the query string.
	issues, resp, err := tp.gh.Search.Issues(ctx, query, nil)
	if err != nil {
		showGHError(err, resp)
		return nil, err
	}
	return issues.Issues, nil
}

func (tp *tagpr) generatenNextLabels(prIssues []*github.Issue) []string {
	majorLabels := tp.cfg.MajorLabels()
	minorLabels := tp.cfg.MinorLabels()
	var nextMinor, nextMajor bool
	for _, issue := range prIssues {
		for _, l := range issue.Labels {
			if slices.Contains(minorLabels, l.GetName()) {
				nextMinor = true
			}
			if slices.Contains(majorLabels, l.GetName()) {
				nextMajor = true
			}
		}
	}
	var nextLabels []string
	if nextMinor {
		nextLabels = append(nextLabels, "tagpr:minor")
	}
	if nextMajor {
		nextLabels = append(nextLabels, "tagpr:major")
	}

	return nextLabels
}

func buildChunkSearchIssuesQuery(qualifiers string, shasStr string) (chunkQueries []string) {
	// Longer than 256 characters are not supported in the query.
	// ref. https://docs.github.com/en/rest/reference/search#limitations-on-query-length
	//
	// However, although not explicitly stated in the documentation, the space separating
	// keywords is counted as one or more characters, so it is possible to exceed 256
	// characters if the text is filled to the very limit of 256 characters.
	// For this reason, the maximum number of chars in the KEYWORD section is limited to
	// the following number.
	const maxKeywordsLength = 200

	// array of SHAs
	keywords := make([]string, 0, 25)
	// Make bulk requests with multiple SHAs of the maximum possible length.
	// If multiple SHAs are specified, the issue search API will treat it like an OR search,
	// and all the pull requests will be searched.
	// This is difficult to read from the current documentation, but that is the current
	// behavior and GitHub support has responded that this is the spec.
	for _, sha := range strings.Split(shasStr, "\n") {
		if strings.TrimSpace(sha) == "" {
			continue
		}
		tempKeywords := append(keywords, sha)
		if len(strings.Join(tempKeywords, " ")) >= maxKeywordsLength {
			chunkQueries = append(chunkQueries, qualifiers+" "+strings.Join(keywords, " "))
			keywords = make([]string, 0, 25)
		}
		keywords = append(keywords, sha)
	}

	if len(keywords) > 0 {
		chunkQueries = append(chunkQueries, qualifiers+" "+strings.Join(keywords, " "))
	}

	return chunkQueries
}
