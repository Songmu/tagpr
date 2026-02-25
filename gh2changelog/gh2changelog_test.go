package gh2changelog_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Songmu/tagpr/gh2changelog"
	"github.com/google/go-github/v83/github"
)

func TestGH2Changelog(t *testing.T) {
	ctx := context.Background()
	gch, err := gh2changelog.New(ctx,
		gh2changelog.Mock(t, []string{"v1.0.1"}, &mockGitter{}, &mockRelGen{}),
		gh2changelog.RepoPath(t.TempDir()))
	if err != nil {
		t.Error(err)
	}

	out, _, err := gch.Draft(ctx, "v1.0.1", "", time.Date(2022, time.September, 3, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Error(err)
	}
	expect := `## [v1.0.1](https://github.com/Songmu/gh2changelog/commits/v1.0.1) - 2022-09-03
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19
`
	if out != expect {
		t.Errorf("expect: %s, but:\n%s", expect, out)
	}

	out, _, err = gch.Latest(ctx)
	if err != nil {
		t.Error(err)
	}
	expect = `## [v1.0.1](https://github.com/Songmu/gh2changelog/commits/v1.0.1) - 2022-08-17
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19
`
	if out != expect {
		t.Errorf("expect:\n%s, but:\n%s", expect, out)
	}

	unreleased, _, err := gch.Unreleased(ctx)
	if err != nil {
		t.Error(err)
	}
	expect = `## [Unreleased](https://github.com/Songmu/gh2changelog/commits/HEAD)
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19
`
	if unreleased != expect {
		t.Errorf("expect:\n%s, but:\n%s", expect, unreleased)
	}

	out, _, err = gch.Changelog(ctx, "v1.0.1")
	if err != nil {
		t.Error(err)
	}
	expect = `## [v1.0.1](https://github.com/Songmu/gh2changelog/commits/v1.0.1) - 2022-08-17
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19
`
	if out != expect {
		t.Errorf("expect: %s, but:\n%s", expect, out)
	}

	outs, _, err := gch.Changelogs(ctx, -1)
	if err != nil {
		t.Error(err)
	}
	out = outs[0]
	expect = `## [v1.0.1](https://github.com/Songmu/gh2changelog/commits/v1.0.1) - 2022-08-17
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19
`
	if out != expect {
		t.Errorf("expect:\n%s, but:\n%s", expect, out)
	}

	out, err = gch.Update(out, gh2changelog.Trunc)
	if err != nil {
		t.Error(err)
	}
	expect = `# Changelog

## [v1.0.1](https://github.com/Songmu/gh2changelog/commits/v1.0.1) - 2022-08-17
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19
`
	if out != expect {
		t.Errorf("expect:\n%s, but:\n%s", expect, out)
	}

	out, err = gch.Update(unreleased, 0)
	if err != nil {
		t.Error(err)
	}
	expect = `# Changelog

## [Unreleased](https://github.com/Songmu/gh2changelog/commits/HEAD)
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19

## [v1.0.1](https://github.com/Songmu/gh2changelog/commits/v1.0.1) - 2022-08-17
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19
`
	if out != expect {
		t.Errorf("expect:\n%s, but:\n%s", expect, out)
	}
}

func TestDraftWithReleaseYamlPath(t *testing.T) {
	ctx := context.Background()
	gch, err := gh2changelog.New(ctx,
		gh2changelog.Mock(t, []string{"v1.0.1"}, &mockGitter{}, &mockRelGen{}),
		gh2changelog.RepoPath(t.TempDir()),
		gh2changelog.ReleaseYamlPath(".github/custom-release.yml"),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, orig, err := gch.Draft(ctx, "v1.0.2", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Body に ConfigurationFilePath が反映されていることを検証
	expect := "<!-- Release notes generated using configuration in .github/custom-release.yml at v1.0.2 -->"
	if !strings.Contains(orig, expect) {
		t.Errorf("expected Body to contain %q, but got:\n%s", expect, orig)
	}
}

type mockRelGen struct {
}

func (mr *mockRelGen) GenerateReleaseNotes(
	ctx context.Context, owner, repo string, opts *github.GenerateNotesOptions) (
	*github.RepositoryReleaseNotes, *github.Response, error) {

	releaseYaml := ".github/release.yml"
	if opts.ConfigurationFilePath != nil {
		releaseYaml = *opts.ConfigurationFilePath
	}
	return &github.RepositoryReleaseNotes{
		Body: fmt.Sprintf(`<!-- Release notes generated using configuration in %[1]s at %[2]s -->
## What's Changed
* add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
* tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19

## New Contributors
* @Songmu made their first contribution in https://github.com/Songmu/gh2changelog/pull/1

**Full Changelog**: https://github.com/Songmu/gh2changelog/commits/%[2]s
`, releaseYaml, opts.TagName),
	}, nil, nil
}

type mockGitter struct {
}

func (mg *mockGitter) Git(args ...string) (string, string, error) {
	table := []struct {
		arg    []string
		result string
	}{{
		[]string{"config", "remote.origin.url"}, "https://github.com/Songmu/gh2changelog.git",
	}, {
		[]string{"log", "-1", "--format=%ai", "--date=iso", "v1.0.1"},
		"2022-08-17 16:03:00 +0900",
	}, {
		[]string{"remote"}, "origin\nSongmu",
	}, {
		[]string{"remote", "show", "origin"}, `* remote origin
  Fetch URL: https://github.com/Songmu/gh2changelog.git
  Push  URL: git@github.com:Songmu/gh2changelog.git
  HEAD branch: main
  Remote branches:
    gitter                                tracked
    main                                  tracked
    refs/remotes/origin/tagpr-from-v0.0.0 stale (use 'git remote prune' to remove)
    tagpr-from-v0.0.1                     tracked
  Local branches configured for 'git pull':
    gitter     merges with remote gitter
    main       merges with remote main
  Local refs configured for 'git push':
    gitter pushes to gitter (fast-forwardable)
    main   pushes to main   (up to date)
`}}

	for _, item := range table {
		if reflect.DeepEqual(args, item.arg) {
			return item.result, "", nil
		}
	}
	return "", "", fmt.Errorf("unsupported args: %v", args)
}
