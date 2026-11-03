package tagpr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v93/github"
	"github.com/k1LoW/calver"
)

func (tp *tagpr) mergedReleasePullRequestForCommit(
	ctx context.Context, commitish string,
) (*github.PullRequest, error) {
	// Retry because GitHub's internal commit-to-PR index may not be updated
	// immediately after a merge, causing the API to return an empty list.
	// This is especially common with squash merges but can also happen with
	// regular merge commits when the workflow triggers within seconds.
	// See https://github.com/Songmu/tagpr/issues/330
	const maxRetries = 3
	const retryInterval = 2 * time.Second

	for i := range maxRetries {
		opts := &github.ListOptions{PerPage: 100}
		foundAssociations := false
		for {
			pulls, resp, err := tp.gh.PullRequests.ListPullRequestsWithCommit(
				ctx, tp.owner, tp.repo, commitish, opts)
			if err != nil {
				tp.showGHError(err, resp)
				return nil, err
			}
			foundAssociations = foundAssociations || len(pulls) > 0
			for _, pr := range pulls {
				if !pr.GetMergedAt().IsZero() &&
					pr.Base != nil &&
					pr.Base.GetRef() == tp.cfg.ReleaseBranch() &&
					tp.isTagPR(pr) {
					return pr, nil
				}
			}
			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
		if foundAssociations {
			return nil, nil
		}
		if i < maxRetries-1 {
			log.Printf("ListPullRequestsWithCommit returned empty for %s, retrying in %s (%d/%d)",
				commitish, retryInterval, i+1, maxRetries)
			time.Sleep(retryInterval)
		}
	}
	return nil, nil
}

func (tp *tagpr) latestMergedReleasePullRequest(ctx context.Context) (*github.PullRequest, error) {
	commitish, _, err := tp.c.Git("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	return tp.mergedReleasePullRequestForCommit(ctx, commitish)
}

const (
	envGitHubEventName = "GITHUB_EVENT_NAME"
	envGitHubEventPath = "GITHUB_EVENT_PATH"
)

// releaseBoundarySHA selects the pre-release boundary used to detect the version
// file and generate release notes without including the release pull request itself.
// Note that "HEAD~" cannot be used for this purpose because it may point to a commit
// of the release pull request itself when "Rebase and merge" was used.
func (tp *tagpr) releaseBoundarySHA(pr *github.PullRequest) (string, error) {
	if os.Getenv(envGitHubEventName) != "push" {
		return mergedBaseSHA(pr)
	}
	headSHA, _, err := tp.c.Git("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	sha, err := pushEventBeforeSHA(
		os.Getenv(envGitHubEventPath),
		"refs/heads/"+tp.cfg.ReleaseBranch(),
		headSHA,
	)
	if err != nil {
		return "", err
	}
	if sha != "" {
		return sha, nil
	}
	return mergedBaseSHA(pr)
}

// pushEventBeforeSHA returns the top-level "before" SHA of the push event payload.
// It returns an empty string when the payload is unavailable or has no usable
// "before", so that the caller can fall back to another source.
func pushEventBeforeSHA(path, releaseRef, headSHA string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read the event payload %q: %w", path, err)
	}

	var payload struct {
		Before string `json:"before"`
		After  string `json:"after"`
		Ref    string `json:"ref"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", fmt.Errorf("failed to parse the event payload %q: %w", path, err)
	}
	if payload.Ref != releaseRef || payload.After != headSHA {
		return "", nil
	}
	if isNullSHA(payload.Before) {
		return "", nil
	}
	return payload.Before, nil
}

// isNullSHA reports whether the SHA is empty or the all-zero SHA, which Git and
// GitHub use to represent "no commit".
func isNullSHA(sha string) bool {
	return sha == "" || strings.Trim(sha, "0") == ""
}

// mergedBaseSHA returns the base SHA of the merged release pull request. It is used
// as a fallback when the push event payload is unavailable. Note that it is the base
// at the time the pull request was last updated, so it may be stale if the release
// branch advanced before the merge. In that case some changes may be missing from
// the release notes, which is acceptable because tagpr is expected to refresh the
// release pull request before it is merged.
func mergedBaseSHA(pr *github.PullRequest) (string, error) {
	if pr == nil {
		return "", errors.New("failed to detect the base commit of the release pull request: no pull request")
	}
	if pr.Base == nil {
		return "", fmt.Errorf(
			"failed to detect the base commit of the release pull request #%d: no base", pr.GetNumber())
	}
	sha := pr.Base.GetSHA()
	if sha == "" {
		return "", fmt.Errorf(
			"failed to detect the base commit of the release pull request #%d: empty base SHA", pr.GetNumber())
	}
	return sha, nil
}

func (tp *tagpr) tagSigningEnabled() (bool, error) {
	value, stderr, err := tp.c.Git("config", "--bool", "--get", "tag.gpgSign")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && value == "" && stderr == "" {
			return false, nil
		}
		return false, fmt.Errorf("failed to read tag.gpgSign: %w", err)
	}
	return value == "true", nil
}

func (tp *tagpr) prepareReleasePlan(
	pr *github.PullRequest, currVer *semv, latestSemverTag string,
) (releasePlan, error) {
	targetSHA, _, err := tp.c.Git("rev-parse", "HEAD")
	if err != nil {
		return releasePlan{}, err
	}
	boundarySHA, err := tp.releaseBoundarySHA(pr)
	if err != nil {
		return releasePlan{}, err
	}
	pendingTag, err := tp.calculateReleaseTag(
		pr, currVer, boundarySHA, targetSHA)
	if err != nil {
		return releasePlan{}, err
	}
	return releasePlan{
		Tag:                     pendingTag,
		CommitSHA:               targetSHA,
		ReleaseNotesBoundarySHA: boundarySHA,
		PullRequestNumber:       pr.GetNumber(),
		BaseTag:                 latestSemverTag,
	}, nil
}

func (tp *tagpr) releaseVersionFile(
	currVer *semv, boundarySHA, targetSHA string,
) (string, error) {
	if tp.cfg.VersionFile() == "-" {
		return "", nil
	}
	if tp.cfg.VersionFile() != "" {
		vfile := strings.TrimSpace(strings.Split(tp.cfg.VersionFile(), ",")[0])
		if vfile == "" {
			return "", nil
		}
		c := *tp.c
		c.outStream = io.Discard
		out, _, err := c.Git("--literal-pathspecs", "ls-tree", "--full-name", "-z",
			targetSHA, "--", vfile)
		if err != nil {
			return "", err
		}
		meta, path, ok := strings.Cut(strings.TrimSuffix(out, "\x00"), "\t")
		fields := strings.Fields(meta)
		if !ok || strings.Contains(path, "\x00") || len(fields) != 3 || fields[1] != "blob" {
			return "", fmt.Errorf("version file %q is not a file in target commit %s", vfile, targetSHA)
		}
		return path, nil
	}
	verReg, err := regexp.Compile(versionRegBase + regexp.QuoteMeta(currVer.Naked()))
	if err != nil {
		return "", err
	}
	c := *tp.c
	c.outStream = io.Discard
	out, _, err := c.Git("grep", "--full-name", "--name-only", "--null",
		"--fixed-strings", "--no-color", "--no-recurse-submodules",
		"-e", currVer.Naked(), boundarySHA, "--")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	var matches []string
	for ref := range strings.SplitSeq(out, "\x00") {
		if ref == "" {
			continue
		}
		path := strings.TrimPrefix(ref, boundarySHA+":")
		parts := strings.Split(path, "/")
		skip := isSkipFile(parts[len(parts)-1])
		for _, dir := range parts[:len(parts)-1] {
			skip = skip || skipDirs[dir]
		}
		if skip {
			continue
		}
		content, _, err := c.Git("show", boundarySHA+":"+path)
		if err != nil {
			return "", err
		}
		if verReg.MatchString(content) {
			matches = append(matches, filepath.ToSlash(path))
		}
	}
	vfile, _ := versionFile(matches)
	return vfile, nil
}

func (tp *tagpr) calculateReleaseTag(
	pr *github.PullRequest,
	currVer *semv,
	boundarySHA, targetSHA string,
) (string, error) {
	vfile, err := tp.releaseVersionFile(currVer, boundarySHA, targetSHA)
	if err != nil {
		return "", err
	}

	var nextTag string
	if vfile != "" {
		c := *tp.c
		c.outStream = io.Discard
		content, _, err := c.Git("show", targetSHA+":"+vfile)
		if err != nil {
			return "", err
		}
		nextVer, err := retrieveVersionFromContents(vfile, []byte(content), currVer)
		if err != nil {
			return "", err
		}
		nextTag = nextVer.Tag()
	} else {
		var labels []string
		for _, l := range pr.Labels {
			labels = append(labels, l.GetName())
		}
		nextTag = currVer.GuessNext(labels).Tag()
	}
	return fullTag(tp.normalizedTagPrefix, nextTag), nil
}

func (tp *tagpr) setPlanOutput(plan releasePlan) error {
	b, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	if _, ok := os.LookupEnv("GITHUB_OUTPUT"); !ok {
		_, err := fmt.Fprintln(tp.out, string(b))
		return err
	}
	return tp.setOutput("plan", string(b))
}

func (tp *tagpr) releasePullRequestBaseTag(pr *github.PullRequest) (string, error) {
	prefix := branchPrefix + branchSafePrefix(tp.normalizedTagPrefix)
	version, ok := strings.CutPrefix(pr.GetHead().GetRef(), prefix)
	if !ok || version == "" {
		return "", fmt.Errorf("failed to determine base tag of pull request #%d", pr.GetNumber())
	}
	tag := fullTag(tp.normalizedTagPrefix, version)
	var err error
	for candidate := version; ; candidate = strings.TrimSuffix(candidate, ".0") {
		tag := fullTag(tp.normalizedTagPrefix, candidate)
		if _, _, err = tp.c.Git("rev-parse", "--verify", "refs/tags/"+tag+"^{commit}"); err == nil {
			return tag, nil
		}
		if !strings.HasSuffix(candidate, ".0") {
			break
		}
	}
	if version == "v0.0.0" || version == "0.0.0" {
		return "", nil
	}
	return "", fmt.Errorf("failed to resolve release pull request base tag %q: %w", tag, err)
}

func (tp *tagpr) fetchReleaseRefs() error {
	out, _, err := tp.c.Git("ls-remote", "--tags", tp.remote())
	if err != nil {
		return err
	}
	tags := parseRemoteTagRefs(out)
	args := []string{"fetch", "--no-tags", tp.remote(),
		"+refs/heads/" + tp.cfg.ReleaseBranch() +
			":refs/remotes/" + tp.remote() + "/" + tp.cfg.ReleaseBranch()}
	var refs []releaseRef
	prefix := "refs/tags/" + tp.normalizedTagPrefix
	for name, sha := range tags {
		ref, ok := tp.classifyReleaseRef(name, sha, prefix)
		if !ok {
			continue
		}
		if ref.version == nil {
			if refspec := tp.releaseFetchRefspec(ref, nil, prefix); refspec != "" {
				args = append(args, refspec)
			}
			continue
		}
		refs = append(refs, ref)
	}
	for _, ref := range refs {
		if refspec := tp.releaseFetchRefspec(ref, refs, prefix); refspec != "" {
			args = append(args, refspec)
		}
	}
	_, _, err = tp.c.Git(args...)
	return err
}

func parseRemoteTagRefs(out string) map[string]string {
	tags := map[string]string{}
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		ref := strings.TrimSuffix(fields[1], "^{}")
		if strings.HasSuffix(fields[1], "^{}") || tags[ref] == "" {
			tags[ref] = fields[0]
		}
	}
	return tags
}

type releaseRef struct {
	name, sha string
	version   *semv
	precision int
}

func (tp *tagpr) classifyReleaseRef(name, sha, prefix string) (releaseRef, bool) {
	version, ok := strings.CutPrefix(name, prefix)
	if !ok {
		return releaseRef{}, false
	}
	if tp.cfg.CalendarVersioning() {
		return releaseRef{name: name, sha: sha},
			tp.validateCalverTag(strings.TrimPrefix(name, "refs/tags/")) == nil
	}
	sv, err := newSemver(version)
	return releaseRef{name, sha, sv, strings.Count(version, ".") + 1}, err == nil
}

func (ref releaseRef) canForceAlias(refs []releaseRef, prefix string) bool {
	if ref.version == nil || ref.precision >= 3 ||
		ref.version.sv.Prerelease() != "" || ref.version.sv.Metadata() != "" {
		return false
	}
	// Only advance aliases whose remote target also has a fully specified tag.
	for _, full := range refs {
		if full.version.Tag() != strings.TrimPrefix(full.name, prefix) ||
			full.sha != ref.sha || full.version.vPrefix != ref.version.vPrefix ||
			full.version.sv.Major() != ref.version.sv.Major() {
			continue
		}
		if ref.precision == 1 || full.version.sv.Minor() == ref.version.sv.Minor() {
			return true
		}
	}
	return false
}

func (tp *tagpr) releaseFetchRefspec(ref releaseRef, refs []releaseRef, prefix string) string {
	localSHA, _, err := tp.c.Git("rev-parse", "--verify", ref.name+"^{commit}")
	if err == nil && localSHA == ref.sha {
		return ""
	}
	force := ""
	if ref.canForceAlias(refs, prefix) {
		force = "+"
	}
	return force + ref.name + ":" + ref.name
}

func (tp *tagpr) applyReleasePlan(ctx context.Context, plan releasePlan) error {
	if err := plan.validate(); err != nil {
		return err
	}
	if err := tp.fetchReleaseRefs(); err != nil {
		return fmt.Errorf("failed to fetch release branch and tags: %w", err)
	}
	if tp.cfg.vPrefix == nil {
		currVer, err := tp.versionFromBaseTag(tp.latestSemverTag())
		if err != nil {
			return err
		}
		if err := tp.cfg.SetVPrefix(currVer.vPrefix); err != nil {
			return err
		}
	}
	targetSHA, err := tp.resolveExactCommit(plan.CommitSHA)
	if err != nil {
		return fmt.Errorf("failed to resolve plan commit SHA: %w", err)
	}
	boundarySHA, err := tp.resolveExactCommit(plan.ReleaseNotesBoundarySHA)
	if err != nil {
		return fmt.Errorf("failed to resolve release notes boundary SHA: %w", err)
	}
	plan.CommitSHA = targetSHA
	plan.ReleaseNotesBoundarySHA = boundarySHA

	if err := tp.requireReleasePlanAncestors(plan); err != nil {
		return err
	}
	pr, err := tp.mergedReleasePullRequestForPlan(ctx, plan)
	if err != nil {
		return err
	}

	_, remoteTagExists, err := tp.inspectExistingTag(
		plan.Tag, targetSHA)
	if err != nil {
		return err
	}
	latestTag := tp.latestSemverTag(plan.Tag)
	if latestTag != plan.BaseTag {
		return fmt.Errorf("base tag changed from %q to %q", plan.BaseTag, latestTag)
	}

	currVer, err := tp.versionFromBaseTag(plan.BaseTag)
	if err != nil {
		return err
	}
	if tp.cfg.CalendarVersioning() {
		if err := tp.validateCalverTag(plan.Tag); err != nil {
			return err
		}
		vfile, err := tp.releaseVersionFile(currVer, boundarySHA, targetSHA)
		if err != nil {
			return err
		}
		if vfile == "" {
			return tp.completeRelease(ctx, plan, pr, remoteTagExists)
		}
	}
	recalculatedTag, err := tp.calculateReleaseTag(
		pr, currVer, boundarySHA, targetSHA)
	if err != nil {
		return err
	}
	if recalculatedTag != plan.Tag {
		return fmt.Errorf(
			"planned tag changed from %q to %q", plan.Tag, recalculatedTag)
	}
	return tp.completeRelease(ctx, plan, pr, remoteTagExists)
}

func (tp *tagpr) requireReleasePlanAncestors(plan releasePlan) error {
	remoteReleaseBranch := tp.remote() + "/" + tp.cfg.ReleaseBranch()
	if err := tp.requireAncestor(plan.CommitSHA, remoteReleaseBranch,
		"target commit is not an ancestor of the release branch"); err != nil {
		return err
	}
	if err := tp.requireAncestor(plan.ReleaseNotesBoundarySHA, plan.CommitSHA,
		"release boundary is not an ancestor of the target commit"); err != nil {
		return err
	}
	if plan.BaseTag != "" {
		if err := tp.requireAncestor(plan.BaseTag, plan.ReleaseNotesBoundarySHA,
			"base tag is not an ancestor of the release boundary"); err != nil {
			return err
		}
	}
	return nil
}

func (tp *tagpr) mergedReleasePullRequestForPlan(
	ctx context.Context, plan releasePlan,
) (*github.PullRequest, error) {
	pr, err := tp.mergedReleasePullRequestForCommit(ctx, plan.CommitSHA)
	if err != nil {
		return nil, err
	}
	if pr == nil || pr.GetNumber() != plan.PullRequestNumber {
		return nil, fmt.Errorf(
			"target commit %s is not associated with merged tagpr pull request #%d",
			plan.CommitSHA, plan.PullRequestNumber)
	}
	if pr.GetMergeCommitSHA() != plan.CommitSHA {
		return nil, fmt.Errorf(
			"target commit %s is not the final merge commit %s of pull request #%d",
			plan.CommitSHA, pr.GetMergeCommitSHA(), plan.PullRequestNumber)
	}
	return pr, nil
}

func (tp *tagpr) validateCalverTag(tag string) error {
	format := tp.cfg.CalendarVersioningFormat()
	if format == "" {
		format = defaultCalendarVersioningFormat
	}
	prefix := tp.normalizedTagPrefix
	vPrefix := strings.HasPrefix(strings.TrimPrefix(tag, prefix), "v")
	if tp.cfg.vPrefix != nil {
		vPrefix = *tp.cfg.vPrefix
	}
	if vPrefix {
		prefix += "v"
	}
	if !strings.HasPrefix(tag, prefix) {
		return fmt.Errorf("planned tag %q must start with %q", tag, prefix)
	}
	version := strings.TrimPrefix(tag, prefix)
	if !vPrefix && strings.HasPrefix(version, "v") {
		return fmt.Errorf("planned tag %q has a v prefix but tagpr.vPrefix is false", tag)
	}
	if _, err := calver.Parse(format, version); err != nil {
		return fmt.Errorf("planned tag %q is not valid for CalVer format %q: %w",
			tag, format, err)
	}
	return nil
}

func (tp *tagpr) completeRelease(
	ctx context.Context,
	plan releasePlan,
	pr *github.PullRequest,
	tagExists bool,
) error {
	var notes *github.RepositoryReleaseNotes
	if tp.cfg.Release() {
		var err error
		notes, err = tp.generateReleaseNotes(ctx, plan, tagExists)
		if err != nil {
			return err
		}
	}
	if !tagExists {
		if err := tp.createAndPushReleaseTag(plan); err != nil {
			return err
		}
	}
	if tp.cfg.Release() {
		if err := tp.ensureGitHubRelease(ctx, plan, notes); err != nil {
			return err
		}
	}
	return tp.setFinalOutputs(plan, pr)
}

func (tp *tagpr) generateReleaseNotes(
	ctx context.Context, plan releasePlan, tagExists bool,
) (*github.RepositoryReleaseNotes, error) {
	previousTag := &plan.BaseTag
	if plan.BaseTag == "" {
		previousTag = nil
	}
	targetCommitish := plan.ReleaseNotesBoundarySHA
	tagName := plan.Tag
	if tagExists {
		tagName += "-notes"
	}
	notes, resp, err := tp.gh.Repositories.GenerateReleaseNotes(
		ctx, tp.owner, tp.repo, github.GenerateNotesRequest{
			TagName:               tagName,
			PreviousTagName:       previousTag,
			TargetCommitish:       &targetCommitish,
			ConfigurationFilePath: new(tp.cfg.ReleaseYAMLPath()),
		})
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}
	if tagExists {
		notes.Name = strings.ReplaceAll(notes.Name, tagName, plan.Tag)
		notes.Body = strings.ReplaceAll(notes.Body, "..."+tagName, "..."+plan.Tag)
		notes.Body = strings.ReplaceAll(notes.Body, "/commits/"+tagName, "/commits/"+plan.Tag)
	}
	return notes, nil
}

func (tp *tagpr) createAndPushReleaseTag(plan releasePlan) error {
	localSHA, _, localErr := tp.c.Git(
		"rev-parse", "--verify", "refs/tags/"+plan.Tag+"^{commit}")
	if localErr != nil {
		tagArgs := []string{"tag"}
		signTag, err := tp.tagSigningEnabled()
		if err != nil {
			return err
		}
		if signTag {
			tagArgs = append(tagArgs, "-s", "-m", "Release "+plan.Tag)
		}
		tagArgs = append(tagArgs, plan.Tag, plan.CommitSHA)
		if _, _, err := tp.c.Git(tagArgs...); err != nil {
			return err
		}
	} else if localSHA != plan.CommitSHA {
		return fmt.Errorf(
			"tag %s already points to %s, want %s",
			plan.Tag, localSHA, plan.CommitSHA)
	}
	ref := "refs/tags/" + plan.Tag
	if _, _, err := tp.c.Git("push", tp.remote(), ref+":"+ref); err != nil {
		_, existsAfterPush, verifyErr := tp.inspectExistingTag(
			plan.Tag, plan.CommitSHA)
		if verifyErr != nil || !existsAfterPush {
			return err
		}
	}
	return nil
}

func (tp *tagpr) ensureGitHubRelease(
	ctx context.Context, plan releasePlan, notes *github.RepositoryReleaseNotes,
) error {
	existingRelease, err := tp.findReleaseByTag(ctx, plan.Tag)
	if err != nil {
		return err
	}
	if existingRelease != nil {
		if existingRelease.GetDraft() != tp.cfg.ReleaseDraft() {
			return fmt.Errorf(
				"GitHub Release for %s already exists with draft=%t, want draft=%t",
				plan.Tag, existingRelease.GetDraft(), tp.cfg.ReleaseDraft())
		}
		return nil
	}
	_, resp, err := tp.gh.Repositories.CreateRelease(
		ctx, tp.owner, tp.repo, github.CreateReleaseRequest{
			TagName:         plan.Tag,
			TargetCommitish: &plan.CommitSHA,
			Name:            &notes.Name,
			Body:            &notes.Body,
			Draft:           new(tp.cfg.ReleaseDraft()),
		})
	if err != nil {
		tp.showGHError(err, resp)
		return err
	}
	return nil
}

func (tp *tagpr) findReleaseByTag(ctx context.Context, tag string) (*github.RepositoryRelease, error) {
	release, resp, err := tp.gh.Repositories.GetReleaseByTag(ctx, tp.owner, tp.repo, tag)
	if err == nil {
		return release, nil
	}
	if resp == nil || resp.StatusCode != 404 {
		tp.showGHError(err, resp)
		return nil, err
	}
	opts := &github.ListOptions{PerPage: 100}
	for {
		releases, resp, err := tp.gh.Repositories.ListReleases(ctx, tp.owner, tp.repo, opts)
		if err != nil {
			tp.showGHError(err, resp)
			return nil, err
		}
		for _, release := range releases {
			if release.GetTagName() == tag {
				return release, nil
			}
		}
		if resp.NextPage == 0 {
			return nil, nil
		}
		opts.Page = resp.NextPage
	}
}

func (tp *tagpr) setFinalOutputs(plan releasePlan, pr *github.PullRequest) error {
	if err := tp.setOutput("tag", plan.Tag); err != nil {
		return err
	}
	if err := tp.setOutput("base_tag", plan.BaseTag); err != nil {
		return err
	}
	b, err := json.Marshal(pr)
	if err != nil {
		return err
	}
	return tp.setOutput("pull_request", string(b))
}

func (tp *tagpr) resolveExactCommit(candidateSHA string) (string, error) {
	sha, _, err := tp.c.Git("rev-parse", "--verify", candidateSHA+"^{commit}")
	if err != nil {
		return "", err
	}
	if sha != candidateSHA {
		return "", fmt.Errorf("%q is not a full commit SHA", candidateSHA)
	}
	return sha, nil
}

func (tp *tagpr) requireAncestor(ancestor, descendant, message string) error {
	if _, _, err := tp.c.Git("merge-base", "--is-ancestor", ancestor, descendant); err != nil {
		return fmt.Errorf("%s: %s is not an ancestor of %s", message, ancestor, descendant)
	}
	return nil
}

func (tp *tagpr) inspectExistingTag(tag, targetSHA string) (bool, bool, error) {
	localSHA, _, localErr := tp.c.Git("rev-parse", "--verify", "refs/tags/"+tag+"^{commit}")
	if localErr == nil && localSHA != targetSHA {
		return false, false,
			fmt.Errorf("tag %s already points to %s, want %s", tag, localSHA, targetSHA)
	}

	out, _, err := tp.c.Git(
		"ls-remote", "--tags", tp.remote(),
		"refs/tags/"+tag, "refs/tags/"+tag+"^{}")
	if err != nil {
		return false, false, err
	}
	if out == "" {
		return localErr == nil, false, nil
	}
	var remoteSHA string
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.HasSuffix(fields[1], "^{}") {
			remoteSHA = fields[0]
			break
		}
		if remoteSHA == "" {
			remoteSHA = fields[0]
		}
	}
	if remoteSHA == "" {
		return false, false, fmt.Errorf("failed to parse remote tag %s", tag)
	}
	if remoteSHA != targetSHA {
		return false, false, fmt.Errorf(
			"remote tag %s already points to %s, want %s", tag, remoteSHA, targetSHA)
	}
	return localErr == nil, true, nil
}

func (tp *tagpr) remote() string {
	if tp.remoteName == "" {
		return "origin"
	}
	return tp.remoteName
}

func (tp *tagpr) versionFromBaseTag(baseTag string) (*semv, error) {
	version := baseTag
	if version == "" {
		version = "v0.0.0"
	} else {
		version = strings.TrimPrefix(version, tp.normalizedTagPrefix)
	}
	currVer, err := newSemver(version)
	if err != nil {
		return nil, err
	}
	if tp.cfg.vPrefix == nil {
		currVer.vPrefix = strings.HasPrefix(version, "v")
	} else {
		currVer.vPrefix = *tp.cfg.vPrefix
	}
	currVer.asCalendarVersion = tp.cfg.CalendarVersioning()
	currVer.calverFormat = tp.cfg.CalendarVersioningFormat()
	return currVer, nil
}

func (tp *tagpr) tagRelease(
	ctx context.Context, pr *github.PullRequest, currVer *semv, latestSemverTag string,
) error {
	plan, err := tp.prepareReleasePlan(pr, currVer, latestSemverTag)
	if err != nil {
		return err
	}
	return tp.completeRelease(ctx, plan, pr, false)
}
