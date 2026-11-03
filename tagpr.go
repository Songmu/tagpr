package tagpr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Songmu/gitconfig"
	"github.com/Songmu/gitsemvers"
	"github.com/google/go-github/v93/github"
	"github.com/k1LoW/calver"
)

const (
	defaultReleaseBranch = "main"
	autoCommitMessage    = "prepare for the next release"
	autoChangelogMessage = "update CHANGELOG.md"
	autoLabelName        = "tagpr"
	branchPrefix         = "tagpr-from-"
	dependabotLogin      = "dependabot[bot]"
)

type tagpr struct {
	c                       *commander
	gh                      *github.Client
	cfg                     *config
	gitPath                 string
	remoteName, owner, repo string
	out                     io.Writer
	normalizedTagPrefix     string
	runOptions              runOptions
}

func (tp *tagpr) latestSemverTag(excludedTags ...string) string {
	// CalVer mode: bypass gitsemvers because it normalizes version strings
	// and loses zero-padding (e.g., v2026.0123.0 becomes invalid)
	if tp.cfg.CalendarVersioning() {
		return tp.latestCalverTag(excludedTags...)
	}

	vers := (&gitsemvers.Semvers{
		GitPath:   tp.gitPath,
		TagPrefix: tp.cfg.TagPrefix(),
	}).VersionStrings()

	fixedMajor, _ := tp.cfg.FixedMajorVersion()

	for _, v := range vers {
		if slices.Contains(excludedTags, v) {
			continue
		}
		semvPart := strings.TrimPrefix(v, tp.normalizedTagPrefix)
		sv, err := newSemver(semvPart)
		if err != nil {
			continue
		}

		// Filter by fixedMajorVersion
		if fixedMajor != nil {
			if sv.sv.Major() != *fixedMajor {
				continue
			}
		}

		// Filter by vPrefix
		if tp.cfg.vPrefix != nil && strings.HasPrefix(semvPart, "v") != *tp.cfg.vPrefix {
			continue
		}
		for version := sv.Tag(); ; version = strings.TrimSuffix(version, ".0") {
			tag := fullTag(tp.normalizedTagPrefix, version)
			if slices.Contains(vers, tag) {
				if slices.Contains(excludedTags, tag) {
					break
				}
				return tag
			}
			if !strings.HasSuffix(version, ".0") {
				break
			}
		}
	}
	return ""
}

func (tp *tagpr) latestCalverTag(excludedTags ...string) string {
	format := tp.cfg.CalendarVersioningFormat()
	if format == "" {
		format = defaultCalendarVersioningFormat
	}
	prefix := tp.normalizedTagPrefix

	out, _, err := tp.c.Git("tag", "-l")
	if err != nil {
		return ""
	}

	type tagCV struct {
		tag string
		cv  *calver.Calver
	}
	var tagCVs []tagCV

	for _, tag := range strings.Split(out, "\n") {
		tag = strings.TrimSpace(tag)
		if tag == "" || slices.Contains(excludedTags, tag) {
			continue
		}

		if prefix != "" && !strings.HasPrefix(tag, prefix) {
			continue
		}

		verPart := strings.TrimPrefix(tag, prefix)
		hasVPrefix := strings.HasPrefix(verPart, "v")
		verStr := strings.TrimPrefix(verPart, "v")

		if tp.cfg.vPrefix != nil && hasVPrefix != *tp.cfg.vPrefix {
			continue
		}

		cv, err := calver.Parse(format, verStr)
		if err != nil {
			continue
		}

		tagCVs = append(tagCVs, tagCV{tag: tag, cv: cv})
	}

	if len(tagCVs) == 0 {
		return ""
	}

	cvs := make(calver.Calvers, len(tagCVs))
	for i, tc := range tagCVs {
		cvs[i] = tc.cv
	}
	latest, err := cvs.Latest()
	if err != nil {
		return ""
	}

	for _, tc := range tagCVs {
		if tc.cv.String() == latest.String() {
			return tc.tag
		}
	}

	return ""
}

func (tp *tagpr) getNextLabels(ctx context.Context, mergedFeatureHeadShas []string, prShasStr, fromCommitish string) ([]string, error) {
	var prIssues []*github.Issue
	for _, number := range mergedPullRequestNumbers(mergedFeatureHeadShas, prShasStr) {
		issue, resp, err := tp.gh.Issues.Get(ctx, tp.owner, tp.repo, number)
		if err != nil {
			tp.showGHError(err, resp)
			return []string{}, err
		}
		prIssues = append(prIssues, issue)
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

	releaseBranch := tp.cfg.ReleaseBranch()

	logArgs := tagLogArgs(tp.normalizedTagPrefix, "log", "--pretty=format:%h", "--abbrev=7", "--no-merges", "--first-parent",
		fmt.Sprintf("%s..%s/%s", fromCommitish, tp.remoteName, releaseBranch))
	shasStr, _, err := tp.c.Git(logArgs...)
	if err != nil {
		return []string{}, err
	}
	queryBase := fmt.Sprintf("repo:%s/%s is:pr is:closed", tp.owner, tp.repo)
	for _, query := range buildChunkSearchIssuesQuery(queryBase, shasStr) {
		tmpIssues, err := tp.searchIssues(ctx, query)
		if err != nil {
			return []string{}, err
		}
		prIssues = append(prIssues, tmpIssues...)
	}

	nextLabels := tp.generateNextLabels(prIssues)

	return nextLabels, nil
}

func (tp *tagpr) initializeReleaseYaml(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(`changelog:
  exclude:
    labels:
      - tagpr
`), 0644); err != nil {
		return err
	}
	_, _, _ = tp.c.Git("add", "-f", path)
	return nil
}

func newTagPR(ctx context.Context, c *commander) (*tagpr, error) {
	tp := &tagpr{
		c: c, gitPath: c.gitPath, out: c.outStream,
		runOptions: runOptions{Mode: executionModeAuto},
	}

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

// isTagPR checks if a PR is a tagpr-generated PR that matches this instance's configuration
func (tp *tagpr) isTagPR(pr *github.PullRequest) bool {
	// Basic validation
	if pr == nil {
		return false
	}
	if pr.Head == nil || pr.Head.Ref == nil {
		return false
	}

	headRef := *pr.Head.Ref

	// Check branch prefix
	if !strings.HasPrefix(headRef, branchPrefix) {
		return false
	}

	// Extract suffix after branch prefix
	suffix := headRef[len(branchPrefix):]

	// Get expected prefix for this tagpr instance
	// branchSafePrefix converts tag prefixes like "gh2changelog/" to "gh2changelog-"
	expectedBranchPrefix := branchSafePrefix(tp.normalizedTagPrefix)

	// Validate prefix matches configuration
	matched := false
	if expectedBranchPrefix == "" {
		// Root module (no prefix): reject branches with module prefixes
		// Find where version begins (first 'v' or digit)
		versionPos := strings.IndexFunc(suffix, func(r rune) bool {
			return r == 'v' || (r >= '0' && r <= '9')
		})
		// Version must start at position 0 (no module prefix before it)
		// If no version marker found, versionPos is -1 and matched stays false
		matched = (versionPos == 0 && !strings.ContainsRune(suffix, '-'))
	} else {
		// Submodule (has prefix): branch must contain our prefix
		matched = strings.HasPrefix(suffix, expectedBranchPrefix)
	}

	if !matched {
		return false
	}

	// Check for tagpr label
	for _, lbl := range pr.Labels {
		if lbl.GetName() == autoLabelName {
			return true
		}
	}

	return false
}

func (tp *tagpr) Run(ctx context.Context) (runErr error) {
	var commandErr error
	defer func() {
		runErr = errors.Join(runErr, commandErr)
	}()
	if err := tp.runOptions.validate(); err != nil {
		return err
	}
	commitMessage := tp.cfg.CommitPrefix() + " " + autoCommitMessage
	changelogMessage := tp.cfg.CommitPrefix() + " " + autoChangelogMessage

	releaseBranch, err := tp.configureReleaseBranch()
	if err != nil {
		return err
	}
	if tp.runOptions.Mode == executionModeApply {
		return tp.applyReleasePlan(ctx, *tp.runOptions.Plan)
	}
	latestSemverTag, fromCommitish, currVer, err := tp.currentReleaseVersion()
	if err != nil {
		return err
	}
	if err := tp.requireReleaseBranch(releaseBranch); err != nil {
		return err
	}
	pr, err := tp.latestMergedReleasePullRequest(ctx)
	if err != nil {
		return err
	}
	if pr != nil {
		return tp.handleMergedRelease(ctx, pr)
	}

	nextLabels, err := tp.mergedChangeLabels(ctx, fromCommitish, releaseBranch)
	if err != nil {
		return err
	}
	ref, resp, err := tp.gh.Git.GetRef(ctx, tp.owner, tp.repo, "refs/heads/"+releaseBranch)
	if err != nil {
		tp.showGHError(err, resp)
		return err
	}
	rcBranch := fmt.Sprintf("%s%s%s", branchPrefix, branchSafePrefix(tp.normalizedTagPrefix), currVer.Tag())
	head := fmt.Sprintf("%s:%s", tp.owner, rcBranch)
	currTagPR, nextVer, addingLabels, err := tp.planReleasePullRequest(ctx, head, releaseBranch, currVer, nextLabels)
	if err != nil {
		return err
	}
	vfiles, err := tp.prepareReleaseChanges(currVer, nextVer, &commandErr)
	if err != nil {
		return err
	}
	newCommit, err := tp.createPreparationCommit(ctx, ref, commitMessage)
	if err != nil {
		return err
	}
	newCommit, ref, err = tp.replayManualReleaseCommits(ctx, releaseBranch, rcBranch, commitMessage, changelogMessage, newCommit, ref)
	if err != nil {
		return err
	}
	nextVer, err = tp.reevaluateReleaseVersion(currVer, nextVer, vfiles)
	if err != nil {
		return err
	}

	draft, prArg := tp.releasePullRequestTemplate(ctx, latestSemverTag, releaseBranch, rcBranch, currVer, nextVer)
	pt := loadPRTmpl(tp.cfg)
	newCommit, err = tp.commitReleaseChangelog(ctx, draft, newCommit, changelogMessage)
	if err != nil {
		return err
	}
	prText, err := pt.Render(prArg)
	if err != nil {
		return err
	}
	if err := tp.updateReleasePullRequestBranchRef(ctx, rcBranch, ref, newCommit); err != nil {
		return err
	}
	return tp.publishReleasePullRequest(ctx, currTagPR, prText, releaseBranch, head, addingLabels)
}

func replaceCompareLink(orig, host, owner, repo, currTag, nextTag, rcBranch string) string {
	const base = `**Full Changelog**: https://%s/%s/%s/compare/%s...%s`
	beforeCompareURL := fmt.Sprintf(base, host, owner, repo, currTag, nextTag)
	afterCompareURL := fmt.Sprintf(base, host, owner, repo, currTag, rcBranch)
	return strings.ReplaceAll(orig, beforeCompareURL, afterCompareURL)
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

func (tp *tagpr) getVfiles(currVer *semv) ([]string, error) {
	var vfiles []string

	if vf := tp.cfg.VersionFile(); vf != "" && vf != "-" {
		vfiles = splitVersionFiles(vf)
	} else if tp.cfg.versionFile == nil {
		vfile, err := detectVersionFile(".", currVer)
		if err != nil {
			return []string{}, err
		}
		if err := tp.cfg.SetVersionFile(vfile); err != nil {
			return []string{}, err
		}
		vfiles = append(vfiles, vfile)
	}

	return vfiles, nil
}

func (tp *tagpr) Exec(name, command string, currVer, nextVer *semv) error {
	prog := command
	var progArgs []string
	if strings.ContainsAny(prog, " \n") {
		progArgs = []string{"-c", prog}
		prog = "sh"
	}
	_, stderr, err := tp.c.Cmd(prog, progArgs, map[string]string{
		"TAGPR_CURRENT_VERSION": currVer.Tag(),
		"TAGPR_NEXT_VERSION":    nextVer.Tag(),
	})
	if err == nil {
		return nil
	}

	commandErr := fmt.Errorf("%s %q failed: %w", name, command, err)
	if stderr != "" {
		commandErr = fmt.Errorf("%w\n%s", commandErr, stderr)
	}
	return commandErr
}

func (tp *tagpr) showReleaseCommandError(name string, err error) {
	fmt.Fprintln(tp.c.errStream, err)
	showErrorAnnotation(tp.c.errStream, name+" failed", err.Error())
}

func (tp *tagpr) runReleaseCommand(errs *error, name, command string, currVer, nextVer *semv) {
	if err := tp.Exec(name, command, currVer, nextVer); err != nil {
		*errs = errors.Join(*errs, err)
		tp.showReleaseCommandError(name, err)
	}
}

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
		tp.showGHError(err, resp)
		return nil, err
	}
	return issues.Issues, nil
}

func isDependabotPR(issue *github.Issue) bool {
	return issue.GetUser().GetLogin() == dependabotLogin
}

func (tp *tagpr) generateNextLabels(prIssues []*github.Issue) []string {
	majorLabels := tp.cfg.MajorLabels()
	minorLabels := tp.cfg.MinorLabels()
	var nextMinor, nextMajor bool
	for _, issue := range prIssues {
		if isDependabotPR(issue) {
			continue
		}
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
	for sha := range strings.SplitSeq(shasStr, "\n") {
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
