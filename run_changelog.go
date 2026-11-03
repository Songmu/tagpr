package tagpr

import (
	"context"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Songmu/tagpr/gh2changelog"
	"github.com/google/go-github/v93/github"
)

func (tp *tagpr) newReleaseDraft(ctx context.Context, latestSemverTag, releaseBranch string, nextVer *semv) *lazyDraftChangelog {
	draft := &lazyDraftChangelog{load: func() (*draftChangelog, error) {
		opts := []gh2changelog.Option{
			gh2changelog.GitPath(tp.gitPath),
			gh2changelog.SetOutputs(tp.c.outStream, tp.c.errStream),
			gh2changelog.GitHubClient(tp.gh),
			gh2changelog.TagPrefix(tp.normalizedTagPrefix),
			gh2changelog.ChangelogMdPath(tp.cfg.ChangelogFile()),
		}
		if tp.cfg.ReleaseYAMLPath() != "" {
			opts = append(opts, gh2changelog.ReleaseYamlPath(tp.cfg.ReleaseYAMLPath()))
		}
		if fixedMajor, err := tp.cfg.FixedMajorVersion(); err == nil && fixedMajor != nil {
			opts = append(opts, gh2changelog.FilteredMajorVersion(*fixedMajor))
		}
		if tp.cfg.CalendarVersioning() && latestSemverTag != "" {
			opts = append(opts, gh2changelog.VersionTags([]string{latestSemverTag}))
		}
		gch, err := gh2changelog.New(ctx, opts...)
		if err != nil {
			return nil, err
		}

		draftNextTag := fullTag(tp.normalizedTagPrefix, nextVer.Tag())
		changelog, generatedNotes, err := gch.Draft(ctx, draftNextTag, releaseBranch, time.Now())
		if err != nil {
			return nil, err
		}
		return &draftChangelog{
			generator:      gch,
			changelog:      changelog,
			generatedNotes: generatedNotes,
		}, nil
	}}
	return draft
}

func (tp *tagpr) releasePullRequestTemplate(ctx context.Context, latestSemverTag, releaseBranch, rcBranch string, currVer, nextVer *semv) (*lazyDraftChangelog, *tmplArg) {
	host := "github.com"
	if baseURL, err := url.Parse(tp.gh.BaseURL()); err == nil && baseURL.Host != "" {
		host = strings.TrimPrefix(baseURL.Host, "api.")
	}
	currTag := fullTag(tp.normalizedTagPrefix, currVer.Tag())
	nextTag := fullTag(tp.normalizedTagPrefix, nextVer.Tag())
	draft := tp.newReleaseDraft(ctx, latestSemverTag, releaseBranch, nextVer)
	prArg := &tmplArg{
		NextVersion: nextVer.Tag(),
		Branch:      rcBranch,
		TagPrefix:   strings.TrimSuffix(tp.normalizedTagPrefix, "/"),
		loadChangelog: func() (string, error) {
			generatedNotes, err := draft.GeneratedNotes()
			if err != nil {
				return "", err
			}
			return replaceCompareLink(
				generatedNotes, host, tp.owner, tp.repo, currTag, nextTag, rcBranch), nil
		},
	}
	return draft, prArg
}

func (tp *tagpr) commitReleaseChangelog(ctx context.Context, draft *lazyDraftChangelog, newCommit *github.Commit, changelogMessage string) (*github.Commit, error) {
	if !tp.cfg.Changelog() {
		return newCommit, nil
	}
	result, err := draft.Load()
	if err != nil {
		return nil, err
	}
	changelogMd := tp.cfg.ChangelogFile()
	changelog := result.changelog
	if !exists(changelogMd) {
		logs, _, err := result.generator.Changelogs(ctx, 20)
		if err != nil {
			return nil, err
		}
		changelog = strings.Join(
			append([]string{changelog}, logs...), "\n")
	}
	if _, err := result.generator.Update(changelog, 0); err != nil {
		return nil, err
	}

	var treeEntries []*github.TreeEntry
	contentBytes, err := os.ReadFile(changelogMd)
	if err != nil {
		return nil, err
	}
	treeEntries = append(treeEntries, &github.TreeEntry{
		Path:    new(changelogMd),
		Type:    new("blob"),
		Content: new(string(contentBytes)),
		Mode:    new("100644"),
	})
	tree, resp, err := tp.gh.Git.CreateTree(ctx, tp.owner, tp.repo, *newCommit.SHA, treeEntries)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}
	commit := github.Commit{
		Message: new(changelogMessage),
		Tree:    tree,
		Parents: []*github.Commit{newCommit},
	}
	newCommit, resp, err = tp.gh.Git.CreateCommit(ctx, tp.owner, tp.repo, commit, nil)
	if err != nil {
		tp.showGHError(err, resp)
		return nil, err
	}
	return newCommit, nil
}
