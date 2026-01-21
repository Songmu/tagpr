package tagpr

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Songmu/gitsemvers"
	"github.com/google/go-github/v74/github"
)

var (
	hasSchemeReg  = regexp.MustCompile("^[^:]+://")
	scpLikeURLReg = regexp.MustCompile("^([^@]+@)?([^:]+):(/?.+)$")
)

var headBranchReg = regexp.MustCompile(`(?m)^\s*HEAD branch: (.*)$`)

func (tp *tagpr) Exec(prog string, currVer, nextVer *semv) {
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

func (tp *tagpr) generateNextLabels(prIssues []*github.Issue) []string {
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

func (tp *tagpr) latestSemverTag() string {
	vers := (&gitsemvers.Semvers{
		GitPath:   tp.gitPath,
		TagPrefix: tp.cfg.TagPrefix(),
	}).VersionStrings()
	if tp.cfg.vPrefix != nil {
		for _, v := range vers {
			// Strip prefix to check vPrefix against semver part
			semvPart := strings.TrimPrefix(v, tp.normalizedTagPrefix)
			if strings.HasPrefix(semvPart, "v") == *tp.cfg.vPrefix {
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

func (tp *tagpr) getNextLabels(ctx context.Context, mergedFeatureHeadShas []string, prShasStr, fromCommitish string) ([]string, error) {
	var prIssues []*github.Issue
	var err error
	for line := range strings.SplitSeq(prShasStr, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		sha, ref := fields[0], fields[1]
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
					return []string{}, err
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

	releaseBranch := tp.cfg.ReleaseBranch()

	logArgs := []string{"log", "--pretty=format:%h", "--abbrev=7", "--no-merges", "--first-parent",
		fmt.Sprintf("%s..%s/%s", fromCommitish, tp.remoteName, releaseBranch)}
	if tp.normalizedTagPrefix != "" {
		logArgs = append(logArgs, "--", strings.TrimSuffix(tp.normalizedTagPrefix, "/"))
	}
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
