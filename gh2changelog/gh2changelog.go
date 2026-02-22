package gh2changelog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Songmu/gitsemvers"
	"github.com/google/go-github/v82/github"
)

type releaseNoteGenerator interface {
	GenerateReleaseNotes(context.Context, string, string, *github.GenerateNotesOptions) (
		*github.RepositoryReleaseNotes, *github.Response, error)
}

// GH2Changelog is to output changelogs
type GH2Changelog struct {
	gitPath           string
	repoPath          string
	tagPrefix         string
	changelogMdPath   string
	releaseYamlPath   *string
	filteredMajorVersion *uint64

	owner, repo, remoteName string
	outStream, errStream    io.Writer
	semvers                 []string

	c   gitter
	gen releaseNoteGenerator
}

// Options is for functional option
type Option func(*GH2Changelog)

// New returns new GH2Changelog
func New(ctx context.Context, opts ...Option) (*GH2Changelog, error) {
	gch := &GH2Changelog{
		gitPath:         "git",
		repoPath:        ".",
		changelogMdPath: defaultChangelogMd,
		outStream:       io.Discard,
		errStream:       io.Discard,
	}
	for _, opt := range opts {
		opt(gch)
	}

	if gch.c == nil {
		gch.c = &commander{
			gitPath:   gch.gitPath,
			dir:       gch.repoPath,
			outStream: gch.outStream,
			errStream: gch.errStream}
	}
	if gch.semvers == nil {
		gch.semvers = (&gitsemvers.Semvers{
			GitPath:   gch.gitPath,
			RepoPath:  gch.repoPath,
			TagPrefix: gch.tagPrefix,
		}).VersionStrings()
	}
	if gch.filteredMajorVersion != nil {
		gch.semvers = filterByMajorVersion(gch.semvers, gch.tagPrefix, *gch.filteredMajorVersion)
	}

	var err error
	gch.remoteName, err = gch.detectRemote()
	if err != nil {
		return nil, err
	}
	remoteURL, _, err := gch.c.Git("config", "remote."+gch.remoteName+".url")
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
	gch.owner = m[0]
	gch.repo = strings.TrimSuffix(m[1], ".git")

	if gch.gen == nil {
		cli, err := ghClient(ctx, "", u.Hostname())
		if err != nil {
			return nil, err
		}
		gch.gen = cli.Repositories
	}

	return gch, nil
}

// Draft gets draft changelog
func (gch *GH2Changelog) Draft(
	ctx context.Context, nextTag, releaseBranch string, releaseDate time.Time) (string, string, error) {
	vers := gch.semvers
	var previousTag *string
	if len(vers) > 0 {
		previousTag = &vers[0]
	}
	if releaseBranch == "" {
		var err error
		releaseBranch, err = gch.defaultBranch()
		if err != nil {
			return "", "", err
		}
	}
	releases, _, err := gch.gen.GenerateReleaseNotes(
		ctx, gch.owner, gch.repo, &github.GenerateNotesOptions{
			TagName:               nextTag,
			PreviousTagName:       previousTag,
			TargetCommitish:       &releaseBranch,
			ConfigurationFilePath: gch.releaseYamlPath,
		})
	if err != nil {
		return "", "", err
	}
	return convertKeepAChangelogFormat(releases.Body, releaseDate), releases.Body, nil
}

// Unreleased gets unreleased changelog
func (gch *GH2Changelog) Unreleased(ctx context.Context) (string, string, error) {
	const tentativeTag = "v999999.999.999"
	body, orig, err := gch.Draft(ctx, tentativeTag, "", time.Now())
	if err != nil {
		return "", "", err
	}
	bodies := strings.Split(body, "\n")
	for i, b := range bodies {
		if strings.HasPrefix(b, `## [`+tentativeTag+`](http`) {
			b = strings.Replace(b, tentativeTag, "Unreleased", 1)
			b = strings.Replace(b, tentativeTag, "HEAD", 1)
			b = strings.TrimRight(b, " -0123456789") // remove date suffix e.g. " - 2022-06-05"
			bodies[i] = b
		}
	}
	return strings.TrimSpace(strings.Join(bodies, "\n")) + "\n", orig, nil
}

// Latest gets latest changelog
func (gch *GH2Changelog) Latest(ctx context.Context) (string, string, error) {
	vers := gch.semvers
	if len(vers) == 0 {
		return "", "", errors.New("no change log found. Never released yet")
	}
	return gch.Changelog(ctx, vers[0])
}

// Changelog gets changelog for specified tag
func (gch *GH2Changelog) Changelog(ctx context.Context, tag string) (string, string, error) {
	date, _, err := gch.c.Git("log", "-1", "--format=%ai", "--date=iso", tag)
	if err != nil {
		return "", "", err
	}
	d, _ := time.Parse("2006-01-02 15:04:05 -0700", date)
	releases, _, err := gch.gen.GenerateReleaseNotes(
		ctx, gch.owner, gch.repo, &github.GenerateNotesOptions{
			TagName: tag,
		})
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(convertKeepAChangelogFormat(releases.Body, d)) + "\n", releases.Body, nil
}

// Changelogs gets changelogs
func (gch *GH2Changelog) Changelogs(ctx context.Context, limit int) ([]string, []string, error) {
	vers := gch.semvers
	var (
		logs     []string
		origLogs []string
	)
	for i, ver := range vers {
		if limit != -1 && i > limit {
			break
		}
		log, orig, err := gch.Changelog(ctx, ver)
		if err != nil {
			return nil, nil, err
		}
		origLogs = append(origLogs, orig)
		logs = append(logs, log)
	}
	return logs, origLogs, nil
}

const (
	DryRun = 1 << iota
	Trunc
)

const (
	defaultChangelogMd = "CHANGELOG.md"
	heading            = "# Changelog\n"
)

// Update CHANGELOG.md
func (gch *GH2Changelog) Update(section string, mode int) (string, error) {
	dryRun := mode&DryRun != 0
	trunc := mode&Trunc != 0
	chMdPath := gch.path()
	out := section

	var orig string
	if !trunc {
		b, err := os.ReadFile(chMdPath)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		orig = string(b)
	}

	if orig == "" {
		out = heading + "\n" + out
	} else {
		out = insertNewChangelog(orig, out)
	}

	if !dryRun {
		if err := os.WriteFile(chMdPath, []byte(out), 0666); err != nil {
			return "", err
		}
	}
	return out, nil
}

func (gch *GH2Changelog) path() string {
	if gch.changelogMdPath != "" {
		return filepath.Join(gch.repoPath, gch.changelogMdPath)
	}
	return filepath.Join(gch.repoPath, defaultChangelogMd)
}

func (gch *GH2Changelog) detectRemote() (string, error) {
	remotesStr, _, err := gch.c.Git("remote")
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

var headBranchReg = regexp.MustCompile(`(?m)^\s*HEAD branch: (.*)$`)

func filterByMajorVersion(vers []string, tagPrefix string, major uint64) []string {
	var filtered []string
	for _, v := range vers {
		s := strings.TrimPrefix(v, tagPrefix)
		s = strings.TrimPrefix(s, "v")
		parts := strings.SplitN(s, ".", 2)
		if len(parts) == 0 {
			continue
		}
		n, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			continue
		}
		if n == major {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func (gch *GH2Changelog) defaultBranch() (string, error) {
	// `git symbolic-ref refs/remotes/origin/HEAD` sometimes doesn't work
	// So use `git remote show origin` for detecting default branch
	show, _, err := gch.c.Git("remote", "show", gch.remoteName)
	if err != nil {
		return "", fmt.Errorf("failed to detect default branch: %w", err)
	}
	m := headBranchReg.FindStringSubmatch(show)
	if len(m) < 2 {
		return "", fmt.Errorf("failed to detect default branch from remote: %s", gch.remoteName)
	}
	return m[1], nil
}
