package gh2changelog_test

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Songmu/tagpr/gh2changelog"
	"github.com/google/go-github/v93/github"
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

func TestMajorVersionFiltering(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		opts   []gh2changelog.Option
		expect []string
	}{
		{
			name:   "major segment and original order",
			tags:   []string{"v2.9.0", "v1.8.0", "2.malformed", "v2", "v2.1.0", "v3.0.0", "", "vx.0.0", "18446744073709551616.0"},
			opts:   []gh2changelog.Option{gh2changelog.FilteredMajorVersion(2)},
			expect: []string{"v2.9.0", "2.malformed", "v2", "v2.1.0"},
		},
		{
			name:   "prefix stripped before optional v",
			tags:   []string{"component/v2.3.0", "component/2.bad-tail", "other/v2.0.0", "component/v1.0.0", "v2.1.0", "component/v2.0.0"},
			opts:   []gh2changelog.Option{gh2changelog.TagPrefix("component/"), gh2changelog.FilteredMajorVersion(2)},
			expect: []string{"component/v2.3.0", "component/2.bad-tail", "v2.1.0", "component/v2.0.0"},
		},
		{
			name:   "zero major",
			tags:   []string{"v0.2.0", "v1.0.0", "0.tail"},
			opts:   []gh2changelog.Option{gh2changelog.FilteredMajorVersion(0)},
			expect: []string{"v0.2.0", "0.tail"},
		},
		{
			name: "last options win before filtering",
			tags: []string{"v9.0.0"},
			opts: []gh2changelog.Option{
				gh2changelog.VersionTags([]string{"pkg/v1.0.0", "pkg/v2.0.0"}),
				gh2changelog.TagPrefix("wrong/"), gh2changelog.TagPrefix("pkg/"),
				gh2changelog.FilteredMajorVersion(1), gh2changelog.FilteredMajorVersion(2),
			},
			expect: []string{"pkg/v2.0.0"},
		},
		{
			name: "no matching major",
			tags: []string{"v1.0.0", "invalid"},
			opts: []gh2changelog.Option{gh2changelog.FilteredMajorVersion(2)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gen := &mockRelGen{}
			opts := append([]gh2changelog.Option{gh2changelog.Mock(t, tt.tags, &mockGitter{}, gen)}, tt.opts...)
			gch, err := gh2changelog.New(ctx, opts...)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := gch.Draft(ctx, "next", "release", time.Time{}); err != nil {
				t.Fatal(err)
			}
			previous := gen.requests[0].PreviousTagName
			if len(tt.expect) == 0 {
				if previous != nil {
					t.Fatalf("previous tag = %q, want nil", *previous)
				}
				if body, original, err := gch.Latest(ctx); body != "" || original != "" ||
					err == nil || err.Error() != "no change log found. Never released yet" {
					t.Fatalf("Latest = %q, %q, %v; want empty outputs and never-released error", body, original, err)
				}
			} else if previous == nil || *previous != tt.expect[0] {
				t.Fatalf("previous tag = %v, want %q", previous, tt.expect[0])
			}
			logs, originals, err := gch.Changelogs(ctx, -1)
			if err != nil {
				t.Fatal(err)
			}
			var tags []string
			for _, req := range gen.requests[1:] {
				tags = append(tags, req.TagName)
			}
			if !reflect.DeepEqual(tags, tt.expect) || len(logs) != len(tags) || len(originals) != len(tags) {
				t.Fatalf("generated tags = %v, want %v; log counts = %d/%d", tags, tt.expect, len(logs), len(originals))
			}
		})
	}
}

func TestNewVersionTags(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", dir},
		{"-C", dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "initial"},
		{"-C", dir, "-c", "tag.gpgSign=false", "tag", "v1.2.3"},
		{"-C", dir, "remote", "add", "origin", "https://github.com/Songmu/gh2changelog.git"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	for _, tt := range []struct {
		name string
		tags []string
		want *string
	}{
		{"nil discovers tags", nil, new("v1.2.3")},
		{"empty bypasses discovery", []string{}, nil},
		{"provided bypasses discovery", []string{"2026.11"}, new("2026.11")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gen := &mockRelGen{}
			gch, err := gh2changelog.New(ctx, gh2changelog.Mock(t, nil, nil, gen),
				gh2changelog.RepoPath(dir), gh2changelog.VersionTags(tt.tags))
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := gch.Draft(ctx, "next", "main", time.Time{}); err != nil {
				t.Fatal(err)
			}
			if got := gen.requests[0].PreviousTagName; !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("previous tag = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRepositoryFailures(t *testing.T) {
	gitErr := errors.New("git failed")
	for _, tt := range []struct {
		name    string
		command string
		result  gitResult
		want    string
	}{
		{"remote command", "remote", gitResult{err: gitErr}, "failed to detect remote: git failed"},
		{"no remote", "remote", gitResult{}, "failed to detect remote"},
		{"remote URL command", "config remote.origin.url", gitResult{err: gitErr}, "git failed"},
		{"invalid remote URL", "config remote.origin.url", gitResult{out: "https://[invalid"}, "failed to parse remote"},
		{"missing repository", "config remote.origin.url", gitResult{out: "https://github.com/owner"}, "failed to detect owner and repo from remote URL"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gch, err := gh2changelog.New(context.Background(),
				gh2changelog.Mock(t, []string{}, &mockGitter{responses: map[string]gitResult{tt.command: tt.result}}, &mockRelGen{}))
			if gch != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("New = %v, %v; want nil, %q", gch, err, tt.want)
			}
		})
	}
}

func TestNewRepositoryIdentity(t *testing.T) {
	gen := &mockRelGen{}
	git := &mockGitter{responses: map[string]gitResult{
		"remote":                 {out: "upstream\nfork"},
		"config remote.fork.url": {out: "git@example.com:another/project.git"},
	}}
	ctx := context.Background()
	gch, err := gh2changelog.New(ctx, gh2changelog.Mock(t, []string{}, git, gen))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := gch.Draft(ctx, "next", "release", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if gen.owner != "another" || gen.repo != "project" {
		t.Fatalf("repository = %s/%s, want another/project", gen.owner, gen.repo)
	}
}

func TestChangelogFailures(t *testing.T) {
	gitErr := errors.New("git failed")
	genErr := errors.New("generator failed")
	for _, tt := range []struct {
		name      string
		responses map[string]gitResult
		genErr    error
		run       func(*gh2changelog.GH2Changelog) (string, string, error)
		want      string
		wrapped   error
	}{
		{"draft branch command", map[string]gitResult{"remote show origin": {err: gitErr}}, nil,
			func(g *gh2changelog.GH2Changelog) (string, string, error) {
				return g.Draft(context.Background(), "next", "", time.Time{})
			}, "failed to detect default branch: git failed", gitErr},
		{"unreleased missing branch", map[string]gitResult{"remote show origin": {out: "no HEAD branch"}}, nil,
			func(g *gh2changelog.GH2Changelog) (string, string, error) {
				return g.Unreleased(context.Background())
			}, "failed to detect default branch from remote: origin", nil},
		{"draft generator", nil, genErr,
			func(g *gh2changelog.GH2Changelog) (string, string, error) {
				return g.Draft(context.Background(), "next", "release", time.Time{})
			}, "generator failed", genErr},
		{"unreleased generator", nil, genErr,
			func(g *gh2changelog.GH2Changelog) (string, string, error) {
				return g.Unreleased(context.Background())
			}, "generator failed", genErr},
		{"changelog date command", map[string]gitResult{"log -1 --format=%ai --date=iso v1.0.1": {err: gitErr}}, nil,
			func(g *gh2changelog.GH2Changelog) (string, string, error) {
				return g.Changelog(context.Background(), "v1.0.1")
			}, "git failed", gitErr},
		{"latest generator", nil, genErr,
			func(g *gh2changelog.GH2Changelog) (string, string, error) {
				return g.Latest(context.Background())
			}, "generator failed", genErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gen := &mockRelGen{err: tt.genErr}
			gch, err := gh2changelog.New(context.Background(),
				gh2changelog.Mock(t, []string{"v1.0.1"}, &mockGitter{responses: tt.responses}, gen))
			if err != nil {
				t.Fatal(err)
			}
			body, original, err := tt.run(gch)
			if body != "" || original != "" || err == nil || err.Error() != tt.want {
				t.Fatalf("result = %q, %q, %v; want empty outputs, %q", body, original, err, tt.want)
			}
			if tt.wrapped != nil && !errors.Is(err, tt.wrapped) {
				t.Fatalf("error %v does not preserve %v", err, tt.wrapped)
			}
			if tt.genErr == nil && len(gen.requests) != 0 {
				t.Fatalf("generator called after Git failure: %v", gen.requests)
			}
		})
	}
}

func TestChangelogsLimitAndFailure(t *testing.T) {
	ctx := context.Background()
	tags := []string{"v1.0.1", "v1.0.0", "v0.9.0"}
	for _, tt := range []struct {
		limit int
		count int
	}{{-1, 3}, {0, 1}, {1, 2}} {
		t.Run(fmt.Sprintf("limit %d", tt.limit), func(t *testing.T) {
			gen := &mockRelGen{}
			gch, err := gh2changelog.New(ctx, gh2changelog.Mock(t, tags, &mockGitter{}, gen))
			if err != nil {
				t.Fatal(err)
			}
			logs, originals, err := gch.Changelogs(ctx, tt.limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(logs) != tt.count || len(originals) != tt.count || len(gen.requests) != tt.count {
				t.Fatalf("counts = %d/%d/%d, want %d", len(logs), len(originals), len(gen.requests), tt.count)
			}
			for i, req := range gen.requests {
				if req.TagName != tags[i] || !strings.Contains(logs[i], "["+tags[i]+"]") ||
					!strings.Contains(originals[i], "at "+tags[i]+" -->") {
					t.Fatalf("output %d does not match tag %q", i, tags[i])
				}
			}
		})
	}
	for _, source := range []string{"git", "generator"} {
		t.Run(source+" failure discards partial results", func(t *testing.T) {
			failure := errors.New(source + " failed")
			gen := &mockRelGen{}
			git := &mockGitter{}
			if source == "git" {
				git.responses = map[string]gitResult{"log -1 --format=%ai --date=iso v1.0.0": {err: failure}}
			} else {
				gen.err, gen.failTag = failure, "v1.0.0"
			}
			gch, err := gh2changelog.New(ctx, gh2changelog.Mock(t, tags, git, gen))
			if err != nil {
				t.Fatal(err)
			}
			logs, originals, err := gch.Changelogs(ctx, -1)
			if logs != nil || originals != nil || err != failure {
				t.Fatalf("result = %v, %v, %v; want nil, nil, %v", logs, originals, err, failure)
			}
			wantCalls := 1
			if source == "generator" {
				wantCalls = 2
			}
			if len(gen.requests) != wantCalls {
				t.Fatalf("generator calls = %d, want %d", len(gen.requests), wantCalls)
			}
		})
	}
}

type mockRelGen struct {
	err      error
	failTag  string
	requests []github.GenerateNotesRequest
	owner    string
	repo     string
}

func (mr *mockRelGen) GenerateReleaseNotes(
	ctx context.Context, owner, repo string, opts github.GenerateNotesRequest) (
	*github.RepositoryReleaseNotes, *github.Response, error) {

	mr.requests = append(mr.requests, opts)
	mr.owner, mr.repo = owner, repo
	if mr.err != nil && (mr.failTag == "" || mr.failTag == opts.TagName) {
		return nil, nil, mr.err
	}
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

type gitResult struct {
	out string
	err error
}

type mockGitter struct {
	responses map[string]gitResult
}

func (mg *mockGitter) Git(args ...string) (string, string, error) {
	if result, ok := mg.responses[strings.Join(args, " ")]; ok {
		return result.out, "", result.err
	}
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
		if len(args) == 5 && reflect.DeepEqual(args[:4], []string{"log", "-1", "--format=%ai", "--date=iso"}) {
			return "2022-08-17 16:03:00 +0900", "", nil
		}
	}
	return "", "", fmt.Errorf("unsupported args: %v", args)
}
