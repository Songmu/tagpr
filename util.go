package tagpr

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/v74/github"
)

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func (tp *tagpr) setOutput(name, value string) error {
	fpath, ok := os.LookupEnv("GITHUB_OUTPUT")
	if !ok {
		return nil
	}
	f, err := os.OpenFile(fpath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s=%s\n", name, value)
	return err
}

func showGHError(err error, resp *github.Response) {
	title := "failed to request GitHub API"
	message := err.Error()
	if resp != nil {
		respInfo := []string{
			fmt.Sprintf("status:%d", resp.StatusCode),
		}
		for name := range resp.Header {
			n := strings.ToLower(name)
			if strings.HasPrefix(n, "x-ratelimit") || n == "x-github-request-id" || n == "retry-after" {
				respInfo = append(respInfo, fmt.Sprintf("%s:%s", n, resp.Header.Get(name)))
			}
		}
		message += " " + strings.Join(respInfo, ", ")
	}
	// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#setting-an-error-message
	fmt.Printf("::error title=%s::%s\n", title, message)
}

// normalizeTagPrefix ensures consistent prefix format (with trailing slash).
// Matches gitsemvers behavior: strings.TrimSuffix(prefix, "/") + "/"
func normalizeTagPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	return strings.TrimSuffix(prefix, "/") + "/"
}

// fullTag returns the tag with prefix (e.g., "tools/v1.2.3").
func fullTag(prefix, tag string) string {
	if prefix == "" {
		return tag
	}
	return prefix + tag
}

// branchSafePrefix converts tag prefix to branch-safe format.
// Replaces slashes with hyphens and ensures trailing hyphen.
// e.g., "tools/" -> "tools-", "backend/api/" -> "backend-api-"
func branchSafePrefix(normalizedPrefix string) string {
	if normalizedPrefix == "" {
		return ""
	}
	// Remove trailing slash and replace remaining slashes with hyphens
	s := strings.TrimSuffix(normalizedPrefix, "/")
	s = strings.ReplaceAll(s, "/", "-")
	return s + "-"
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

func replaceCompareLink(orig, host, owner, repo, currTag, nextTag, rcBranch string) string {
	const base = `**Full Changelog**: https://%s/%s/%s/compare/%s...%s`
	beforeCompareURL := fmt.Sprintf(base, host, owner, repo, currTag, nextTag)
	afterCompareURL := fmt.Sprintf(base, host, owner, repo, currTag, rcBranch)
	return strings.ReplaceAll(orig, beforeCompareURL, afterCompareURL)
}

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
