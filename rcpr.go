package rcpr

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/Songmu/gitsemvers"
	"github.com/google/go-github/v45/github"
	"github.com/saracen/walker"
)

const (
	cmdName              = "rcpr"
	gitUser              = "github-actions[bot]"
	gitEmail             = "github-actions[bot]@users.noreply.github.com"
	defaultReleaseBranch = "main"
	autoCommitMessage    = "[rcpr] prepare for the next release"
	autoChangelogMessage = "[rcpr] update CHANGELOG.md"
	autoLableName        = "rcpr"
)

func printVersion(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%s v%s (rev:%s)\n", cmdName, version, revision)
	return err
}

type rcpr struct {
	c                       *commander
	gh                      *github.Client
	remoteName, owner, repo string
}

func (rp *rcpr) latestSemverTag() string {
	vers := (&gitsemvers.Semvers{}).VersionStrings()
	if len(vers) > 0 {
		return vers[0]
	}
	return ""
}

func (rp *rcpr) initialize(ctx context.Context) error {
	var err error
	rp.remoteName, err = rp.detectRemote()
	if err != nil {
		return err
	}
	remoteURL, _, err := rp.c.gitE("config", "remote."+rp.remoteName+".url")
	if err != nil {
		return err
	}
	u, err := parseGitURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to parse remote")
	}
	m := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(m) < 2 {
		return fmt.Errorf("failed to detect owner and repo from remote URL")
	}
	rp.owner = m[0]
	repo := m[1]
	if u.Scheme == "ssh" || u.Scheme == "git" {
		repo = strings.TrimSuffix(repo, ".git")
	}
	rp.repo = repo

	cli, err := ghClient(ctx, "", u.Hostname())
	if err != nil {
		return err
	}
	rp.gh = cli

	isShallow, _, err := rp.c.gitE("rev-parse", "--is-shallow-repository")
	if err != nil {
		return err
	}
	if isShallow == "true" {
		if _, _, err := rp.c.gitE("fetch", "--unshallow"); err != nil {
			return err
		}
	}
	return nil
}

func isRcpr(pr *github.PullRequest) bool {
	for _, label := range pr.Labels {
		if label.GetName() == autoLableName {
			return true
		}
	}
	return false
}

// Run the rcpr
func Run(ctx context.Context, argv []string, outStream, errStream io.Writer) error {
	log.SetOutput(errStream)
	fs := flag.NewFlagSet(
		fmt.Sprintf("%s (v%s rev:%s)", cmdName, version, revision), flag.ContinueOnError)
	fs.SetOutput(errStream)
	ver := fs.Bool("version", false, "display version")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if *ver {
		return printVersion(outStream)
	}

	// main logic follows
	rp := &rcpr{
		c: &commander{outStream: outStream, errStream: errStream, dir: "."},
	}
	if err := rp.initialize(ctx); err != nil {
		return err
	}

	latestSemverTag := rp.latestSemverTag()
	currVerStr := latestSemverTag
	if currVerStr == "" {
		currVerStr = "v0.0.0"
	}
	currVer, err := newSemver(currVerStr)
	if err != nil {
		return err
	}
	// XXX: Do I need to take care of past tags with and without v-prefixes?
	// It might be good to be able to enforce presence or absence in a configuration file item.

	releaseBranch, _ := rp.defaultBranch() // TODO: make release branch configable
	if releaseBranch == "" {
		releaseBranch = defaultReleaseBranch
	}
	branch, _, err := rp.c.gitE("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to git symbolic-ref: %w", err)
	}
	if branch != releaseBranch {
		return fmt.Errorf("you are not on release branch %q, current branch is %q",
			releaseBranch, branch)
	}

	// XXX: should care GIT_*_NAME etc?
	if _, _, err := rp.c.gitE("config", "user.email"); err != nil {
		rp.c.git("config", "--local", "user.email", gitEmail)
	}
	if _, _, err := rp.c.gitE("config", "user.name"); err != nil {
		rp.c.git("config", "--local", "user.name", gitUser)
	}

	{
		// tag and exit if the HEAD is the merged rcpr
		commitish, _, err := rp.c.gitE("rev-parse", "HEAD")
		if err != nil {
			return err
		}
		pulls, _, err := rp.gh.PullRequests.ListPullRequestsWithCommit(
			ctx, rp.owner, rp.repo, commitish, nil)
		if err != nil {
			return err
		}
		if len(pulls) > 0 && isRcpr(pulls[0]) {
			rp.c.git("checkout", "HEAD~")
			vfile, err := detectVersionFile(".", currVer)
			if err != nil {
				return err
			}
			rp.c.git("checkout", releaseBranch)

			var nextTag string
			if vfile != "" {
				nextVer, err := retrieveVersionFromFile(vfile, currVer.vPrefix)
				if err != nil {
					return err
				}
				nextTag = nextVer.Tag()
			} else {
				nextTag = guessNextSemver(currVer, pulls[0]).Tag()
			}
			previousTag := &latestSemverTag
			if *previousTag == "" {
				previousTag = nil
			}

			// To avoid putting pull requests created by rcpr itself in the release notes,
			// we generate release notes in advance.
			// Get the previous commitish to avoid picking up the merge of the pull
			// request made by rcpr.
			targetCommitish, _, err := rp.c.gitE("rev-parse", "HEAD~")
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

			rp.c.git("tag", nextTag)
			if rp.c.err != nil {
				return rp.c.err
			}
			_, _, err = rp.c.gitE("push", "--tags")
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
	}

	rcBranch := fmt.Sprintf("rcpr-%s", currVer.Tag())
	rp.c.gitE("branch", "-D", rcBranch)
	rp.c.git("checkout", "-b", rcBranch)

	head := fmt.Sprintf("%s:%s", rp.owner, rcBranch)
	pulls, _, err := rp.gh.PullRequests.List(ctx, rp.owner, rp.repo,
		&github.PullRequestListOptions{
			Head: head,
			Base: releaseBranch,
		})
	if err != nil {
		return err
	}
	var currRcpr *github.PullRequest
	if len(pulls) > 0 {
		currRcpr = pulls[0]
	}

	nextVer := guessNextSemver(currVer, currRcpr)

	// TODO: make configurable version file
	vfile, err := detectVersionFile(".", currVer)
	if err != nil {
		return err
	}
	if vfile != "" {
		if err := bumpVersionFile(vfile, currVer, nextVer); err != nil {
			return err
		}
	}

	// TODO To be able to run some kind of change script set by configuration in advance.

	rp.c.git("commit", "--allow-empty", "-am", autoCommitMessage)

	// cherry-pick if the remote branch is exists and changed
	// XXX: Do I need to apply merge commits too?
	//     (We ommited merge commits for now, because if we cherry-pick them, we need to add options like "-m 1".
	out, _, err := rp.c.gitE(
		"log", "--no-merges", "--pretty=format:%h %s", "main.."+rp.remoteName+"/"+rcBranch)
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
				_, _, err := rp.c.gitE(
					"cherry-pick", "--keep-redundant-commits", "--allow-empty", commitish)

				// conflict, etc. / Need error handling in case of non-conflict error?
				if err != nil {
					rp.c.gitE("cherry-pick", "--abort")
				}
			}
		}
	}

	if vfile != "" {
		nVer, _ := retrieveVersionFromFile(vfile, nextVer.vPrefix)
		if nVer != nil && nVer.Naked() != nextVer.Naked() {
			nextVer = nVer
		}
	}
	previousTag := &latestSemverTag
	if *previousTag == "" {
		previousTag = nil
	}
	releases, _, err := rp.gh.Repositories.GenerateReleaseNotes(
		ctx, rp.owner, rp.repo, &github.GenerateNotesOptions{
			TagName:         nextVer.Tag(),
			PreviousTagName: previousTag,
			TargetCommitish: &releaseBranch,
		})
	if err != nil {
		return err
	}

	changelog := convertKeepAChangelogFormat(releases.Body, time.Now())
	changelogMd := "CHANGELOG.md"

	var content string
	if exists(changelogMd) {
		byt, err := os.ReadFile(changelogMd)
		if err != nil {
			return err
		}
		content = strings.TrimSpace(string(byt)) + "\n"
	}

	// If the changelog is not in "keep a changelog" format, or if the file does not exist, re-create everything. Is it rough...?
	if !changelogReg.MatchString(content) {
		// We are concerned that depending on the release history, API requests may become more frequent.
		vers := (&gitsemvers.Semvers{}).VersionStrings()
		logs := []string{"# Changelog\n"}
		for i, ver := range vers {
			if i > 10 {
				break
			}
			date, _, _ := rp.c.gitE("log", "-1", "--format=%ai", "--date=iso", ver)
			d, _ := time.Parse("2006-01-02 15:04:05 -0700", date)
			releases, _, _ := rp.gh.Repositories.GenerateReleaseNotes(
				ctx, rp.owner, rp.repo, &github.GenerateNotesOptions{
					TagName: ver,
				})
			logs = append(logs, strings.TrimSpace(convertKeepAChangelogFormat(releases.Body, d))+"\n")
		}
		content = strings.Join(logs, "\n")
	}

	content = insertNewChangelog(content, changelog)
	if err := os.WriteFile(changelogMd, []byte(content), 0644); err != nil {
		return err
	}
	rp.c.gitE("add", changelogMd)
	rp.c.gitE("commit", "-m", autoChangelogMessage)

	if _, _, err := rp.c.gitE("push", "--force", rp.remoteName, rcBranch); err != nil {
		return err
	}

	// TODO: pull request template
	title := fmt.Sprintf("release %s", nextVer.Tag())

	if currRcpr == nil {
		pr, _, err := rp.gh.PullRequests.Create(ctx, rp.owner, rp.repo, &github.NewPullRequest{
			Title: github.String(title),
			Body:  github.String(releases.Body),
			Base:  &releaseBranch,
			Head:  github.String(head),
		})
		if err != nil {
			return err
		}
		_, _, err = rp.gh.Issues.AddLabelsToIssue(
			ctx, rp.owner, rp.repo, *pr.Number, []string{autoLableName})
		return err
	}
	currRcpr.Title = github.String(title)
	currRcpr.Body = github.String(mergeBody(*currRcpr.Body, releases.Body))
	_, _, err = rp.gh.PullRequests.Edit(ctx, rp.owner, rp.repo, *currRcpr.Number, currRcpr)
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
	return update
}

var headBranchReg = regexp.MustCompile(`(?m)^\s*HEAD branch: (.*)$`)

func (rp *rcpr) defaultBranch() (string, error) {
	// `git symbolic-ref refs/remotes/origin/HEAD` sometimes doesn't work
	// So use `git remote show origin` for detecting default branch
	show, _, err := rp.c.gitE("remote", "show", rp.remoteName)
	if err != nil {
		return "", fmt.Errorf("failed to detect defaut branch: %w", err)
	}
	m := headBranchReg.FindStringSubmatch(show)
	if len(m) < 2 {
		return "", fmt.Errorf("failed to detect default branch from remote: %s", rp.remoteName)
	}
	return m[1], nil
}

func (rp *rcpr) detectRemote() (string, error) {
	remotesStr, _, err := rp.c.gitE("remote")
	if err != nil {
		return "", fmt.Errorf("failed to detect remote: %s", err)
	}
	remotes := strings.Fields(remotesStr)
	if len(remotes) == 1 {
		return remotes[0], nil
	}
	for _, r := range remotes {
		if r == "origin" {
			return r, nil
		}
	}
	return "", errors.New("failed to detect remote")
}

const versionRegBase = `(?i)((?:^|[^-_0-9a-zA-Z])version[^-_0-9a-zA-Z].{0,20})`

var versionReg = regexp.MustCompile(versionRegBase + `([0-9]+\.[0-9]+\.[0-9]+)`)

func detectVersionFile(root string, ver *semv) (string, error) {
	verReg, err := regexp.Compile(versionRegBase + regexp.QuoteMeta(ver.Naked()))
	if err != nil {
		return "", err
	}

	errorCb := func(fpath string, err error) error {
		// When running a rcpr binary under the repository root, "text file busy" occurred,
		// so I did error handling as this, but it did not solve the problem, and it is a special case,
		// so we may not need to do the check in particular.
		if os.IsPermission(err) || errors.Is(err, syscall.ETXTBSY) {
			return nil
		}
		return err
	}

	fl := &fileList{}
	if err := walker.Walk(root, func(fpath string, fi os.FileInfo) error {
		if fi.IsDir() {
			// The "testdata" directory is ommited because of the test code for rcpr itself
			if fi.Name() == ".git" || fi.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		joinedPath := filepath.Join(root, fpath)
		bs, err := os.ReadFile(joinedPath)
		if err != nil {
			return errorCb(fpath, err)
		}
		if verReg.Match(bs) {
			fl.append(joinedPath)
		}
		return nil
	}, walker.WithErrorCallback(errorCb)); err != nil {
		return "", err
	}
	list := fl.list()
	if len(list) < 1 {
		return "", nil
	}
	return list[0], nil
	// XXX: Currently, version file detection methods are inaccurate; it might be better to limit it to
	// gemspec, setup.py, setup.cfg, package.json, META.json, and so on. However, there may be cases
	// where some projects have their own version files, and it is annoying to deal with various
	// languages, etc. one by one, so this is the way to go. We would improve it.
}

type fileList struct {
	l  []string
	mu sync.RWMutex
}

func (fl *fileList) append(fpath string) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.l = append(fl.l, fpath)
}

func (fl *fileList) list() []string {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	return fl.l
}

func bumpVersionFile(fpath string, from, to *semv) error {
	verReg, err := regexp.Compile(versionRegBase + regexp.QuoteMeta(from.Naked()))
	if err != nil {
		return err
	}
	bs, err := os.ReadFile(fpath)
	if err != nil {
		return err
	}

	replaced := false
	updated := verReg.ReplaceAllFunc(bs, func(match []byte) []byte {
		if replaced {
			return match
		}
		replaced = true
		return verReg.ReplaceAll(match, []byte(`${1}`+to.Naked()))
	})
	return os.WriteFile(fpath, updated, 0666)
}

func retrieveVersionFromFile(fpath string, vPrefix bool) (*semv, error) {
	bs, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	m := versionReg.FindSubmatch(bs)
	if len(m) < 3 {
		return nil, fmt.Errorf("no version detected from file: %s", fpath)
	}
	ver := string(m[2])
	if vPrefix {
		ver = "v" + ver
	}
	return newSemver(ver)
}

func guessNextSemver(ver *semv, pr *github.PullRequest) *semv {
	var isMajor, isMinor bool
	if pr != nil {
		for _, l := range pr.Labels {
			switch l.GetName() {
			case autoLableName + ":major":
				isMajor = true
			case autoLableName + ":minor":
				isMinor = true
			}
		}
	}

	var nextv semver.Version
	switch {
	case isMajor:
		nextv = ver.v.IncMajor()
	case isMinor:
		nextv = ver.v.IncMinor()
	default:
		nextv = ver.v.IncPatch()
	}

	return &semv{
		v:       &nextv,
		vPrefix: ver.vPrefix,
	}
}
