package tagpr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/Songmu/gh2changelog"
	"github.com/Songmu/gitsemvers"
	"github.com/google/go-github/v47/github"
)

const (
	gitUser              = "github-actions[bot]"
	gitEmail             = "github-actions[bot]@users.noreply.github.com"
	defaultReleaseBranch = "main"
	autoCommitMessage    = "[tagpr] prepare for the next release"
	autoChangelogMessage = "[tagpr] update CHANGELOG.md"
	autoLableName        = "tagpr"
)

type tagpr struct {
	c                       *commander
	gh                      *github.Client
	cfg                     *config
	gitPath                 string
	remoteName, owner, repo string
}

func (tp *tagpr) latestSemverTag() string {
	vers := (&gitsemvers.Semvers{GitPath: tp.gitPath}).VersionStrings()
	if len(vers) > 0 {
		return vers[0]
	}
	return ""
}

func newTagPR(ctx context.Context, c *commander) (*tagpr, error) {
	tp := &tagpr{c: c, gitPath: c.gitPath}

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
	if pr == nil {
		return false
	}
	for _, label := range pr.Labels {
		if label.GetName() == autoLableName {
			return true
		}
	}
	return false
}

func (tp *tagpr) Run(ctx context.Context) error {
	latestSemverTag := tp.latestSemverTag()
	currVerStr := latestSemverTag
	if currVerStr == "" {
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

	var releaseBranch string
	if r := tp.cfg.ReleaseBranch(); r != nil {
		releaseBranch = r.String()
	}
	if releaseBranch == "" {
		releaseBranch, _ = tp.defaultBranch()
		if releaseBranch == "" {
			releaseBranch = defaultReleaseBranch
		}
		if err := tp.cfg.SetRelaseBranch(releaseBranch); err != nil {
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

	// XXX: should care GIT_*_NAME etc?
	if _, _, err := tp.c.Git("config", "user.email"); err != nil {
		if _, _, err := tp.c.Git("config", "--local", "user.email", gitEmail); err != nil {
			return err
		}
	}
	if _, _, err := tp.c.Git("config", "user.name"); err != nil {
		if _, _, err := tp.c.Git("config", "--local", "user.name", gitUser); err != nil {
			return err
		}
	}

	// If the latest commit is a merge commit of the pull request by tagpr,
	// tag the semver to the commit and create a release and exit.
	if pr, err := tp.latestPullRequest(ctx); err != nil || isTagPR(pr) {
		if err != nil {
			return err
		}
		return tp.tagRelease(ctx, pr, currVer, latestSemverTag)
	}

	rcBranch := fmt.Sprintf("tagpr-from-%s", currVer.Tag())
	tp.c.Git("branch", "-D", rcBranch)
	if _, _, err := tp.c.Git("checkout", "-b", rcBranch); err != nil {
		return err
	}

	head := fmt.Sprintf("%s:%s", tp.owner, rcBranch)
	pulls, _, err := tp.gh.PullRequests.List(ctx, tp.owner, tp.repo,
		&github.PullRequestListOptions{
			Head: head,
			Base: releaseBranch,
		})
	if err != nil {
		return err
	}

	var (
		labels    []*github.Label
		currTagPR *github.PullRequest
	)
	if len(pulls) > 0 {
		currTagPR = pulls[0]
		labels = currTagPR.Labels
	}
	nextVer := currVer.GuessNext(labels)

	var vfiles []string
	if vf := tp.cfg.VersionFile(); vf != nil {
		vfiles = strings.Split(vf.String(), ",")
		for i, v := range vfiles {
			vfiles[i] = strings.TrimSpace(v)
		}
	} else {
		vfile, err := detectVersionFile(".", currVer)
		if err != nil {
			return err
		}
		if err := tp.cfg.SetVersionFile(vfile); err != nil {
			return err
		}
		vfiles = []string{vfile}
	}

	if com := tp.cfg.Command(); com != nil {
		prog := com.String()
		var progArgs []string
		if strings.ContainsAny(prog, " \n") {
			prog = "sh"
			progArgs = []string{"-c", prog}
		}
		tp.c.Cmd(prog, progArgs...)
	}

	if vfiles[0] != "" {
		for _, vfile := range vfiles {
			if err := bumpVersionFile(vfile, currVer, nextVer); err != nil {
				return err
			}
		}
	}
	tp.c.Git("add", "-f", tp.cfg.conf) // ignore any errors

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

	if _, _, err := tp.c.Git("commit", "--allow-empty", "-am", autoCommitMessage); err != nil {
		return err
	}

	// cherry-pick if the remote branch is exists and changed
	// XXX: Do I need to apply merge commits too?
	//     (We ommited merge commits for now, because if we cherry-pick them, we need to add options like "-m 1".
	out, _, err := tp.c.Git(
		"log", "--no-merges", "--pretty=format:%h %s", "main.."+tp.remoteName+"/"+rcBranch)
	if err == nil {
		var cherryPicks []string
		for _, line := range strings.Split(out, "\n") {
			m := strings.SplitN(line, " ", 2)
			if len(m) < 2 {
				continue
			}
			commitish := m[0]
			subject := strings.TrimSpace(m[1])
			if subject != autoCommitMessage && subject != autoChangelogMessage {
				cherryPicks = append(cherryPicks, commitish)
			}
		}
		if len(cherryPicks) > 0 {
			// Specify a commitish one by one for cherry-pick instead of multiple commitish,
			// and apply it as much as possible.
			for i := len(cherryPicks) - 1; i >= 0; i-- {
				commitish := cherryPicks[i]
				_, _, err := tp.c.Git(
					"cherry-pick", "--keep-redundant-commits", "--allow-empty", commitish)

				// conflict, etc. / Need error handling in case of non-conflict error?
				if err != nil {
					tp.c.Git("cherry-pick", "--abort")
				}
			}
		}
	}

	// Reread the configuration file (.tagpr) as it may have been rewritten during the cherry-pick process.
	tp.cfg.Reload()
	if tp.cfg.VersionFile() != nil {
		vfiles = strings.Split(tp.cfg.VersionFile().String(), ",")
		for i, v := range vfiles {
			vfiles[i] = strings.TrimSpace(v)
		}
	}
	if vfiles[0] != "" {
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

	changelogMd := "CHANGELOG.md"
	changelog, orig, err := gch.Draft(ctx, nextVer.Tag(), time.Now())
	if err != nil {
		return err
	}
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

	tp.c.Git("add", changelogMd)
	tp.c.Git("commit", "-m", autoChangelogMessage)

	if _, _, err := tp.c.Git("push", "--force", tp.remoteName, rcBranch); err != nil {
		return err
	}

	var tmpl *template.Template
	if t := tp.cfg.Template(); t != nil {
		tmpTmpl, err := template.ParseFiles(t.String())
		if err == nil {
			tmpl = tmpTmpl
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
		pr, _, err := tp.gh.PullRequests.Create(ctx, tp.owner, tp.repo, &github.NewPullRequest{
			Title: github.String(title),
			Body:  github.String(body),
			Base:  &releaseBranch,
			Head:  github.String(head),
		})
		if err != nil {
			return err
		}
		_, _, err = tp.gh.Issues.AddLabelsToIssue(
			ctx, tp.owner, tp.repo, *pr.Number, []string{autoLableName})
		return err
	}
	currTagPR.Title = github.String(title)
	currTagPR.Body = github.String(mergeBody(*currTagPR.Body, body))
	_, _, err = tp.gh.PullRequests.Edit(ctx, tp.owner, tp.repo, *currTagPR.Number, currTagPR)
	return err
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
		return "", fmt.Errorf("failed to detect defaut branch: %w", err)
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
