package tagpr

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v83/github"
)

func TestMergedBaseSHA(t *testing.T) {
	tests := map[string]struct {
		pr      *github.PullRequest
		want    string
		wantErr bool
	}{
		"valid base SHA": {
			pr: &github.PullRequest{
				Base: &github.PullRequestBranch{SHA: github.Ptr("deadbeef")},
			},
			want: "deadbeef",
		},
		"nil pull request": {
			pr:      nil,
			wantErr: true,
		},
		"missing base": {
			pr:      &github.PullRequest{Number: github.Ptr(10)},
			wantErr: true,
		},
		"empty base SHA": {
			pr: &github.PullRequest{
				Number: github.Ptr(10),
				Base:   &github.PullRequestBranch{SHA: github.Ptr("")},
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := mergedBaseSHA(tt.pr)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("mergedBaseSHA() expected an error, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("mergedBaseSHA() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("mergedBaseSHA() = %q, want %q", got, tt.want)
			}
		})
	}
}

// testRepo is a git repository for testing tagRelease with each merge method.
type testRepo struct {
	dir string
	t   *testing.T
}

func (r *testRepo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *testRepo) commit(fname, content, message string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, fname), []byte(content), 0644); err != nil {
		r.t.Fatal(err)
	}
	r.git("add", fname)
	r.git("commit", "-m", message)
}

// newTestRepo creates a repository which has a release branch "main" with a
// v0.1.0 tag and an unmerged release pull request branch with two commits.
// It also prepares a bare remote so that "git push --tags" works.
func newTestRepo(t *testing.T, versionFileContent string) *testRepo {
	t.Helper()
	dir := t.TempDir()
	remote := t.TempDir()

	r := &testRepo{dir: dir, t: t}
	r.git("init", "--initial-branch=main")
	r.git("config", "user.name", "Test")
	r.git("config", "user.email", "test@example.com")
	// keep the fixture independent of the global git configuration
	r.git("config", "merge.ff", "true")
	r.git("config", "commit.gpgsign", "false")
	r.git("config", "tag.gpgsign", "false")
	r.commit("README.md", "# test\n", "initial commit")
	if versionFileContent != "" {
		r.commit("version.go", versionFileContent, "add version file")
	}
	r.git("tag", "v0.1.0")
	r.git("init", "--bare", remote)
	r.git("remote", "add", "origin", remote)
	r.git("push", "-u", "origin", "main")

	r.git("checkout", "-b", "tagpr-from-v0.1.0")
	r.commit("CHANGELOG.md", "# Changelog\n", "[tagpr] update CHANGELOG.md")
	if versionFileContent != "" {
		r.commit("version.go", strings.Replace(versionFileContent, "0.1.0", "0.2.0", 1),
			"[tagpr] prepare for the next release")
	} else {
		r.commit("NEXT.md", "next\n", "[tagpr] prepare for the next release")
	}
	r.git("checkout", "main")
	return r
}

// merge merges the release pull request branch into main by the specified method
// and returns the base SHA, that is, the tip of main just before the merge.
func (r *testRepo) merge(method string) string {
	r.t.Helper()
	baseSHA := r.git("rev-parse", "HEAD")
	switch method {
	case "merge":
		r.git("merge", "--no-ff", "-m", "Merge pull request #1", "tagpr-from-v0.1.0")
	case "squash":
		r.git("merge", "--squash", "tagpr-from-v0.1.0")
		r.git("commit", "-m", "[tagpr] prepare for the next release (#1)")
	case "rebase":
		r.git("rebase", "main", "tagpr-from-v0.1.0")
		r.git("checkout", "main")
		r.git("merge", "--ff-only", "tagpr-from-v0.1.0")
	default:
		r.t.Fatalf("unknown merge method: %s", method)
	}
	return baseSHA
}

// advanceMain adds a commit to main after the release pull request branch was
// created, which makes the base SHA of the pull request stale.
func (r *testRepo) advanceMain(fname, content, message string) string {
	r.t.Helper()
	current := r.git("symbolic-ref", "--short", "HEAD")
	r.git("checkout", "main")
	r.commit(fname, content, message)
	sha := r.git("rev-parse", "HEAD")
	if current != "main" {
		r.git("checkout", current)
	}
	return sha
}

// writeEventFile writes a push event payload and returns its path.
func writeEventFile(t *testing.T, payload string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(path, []byte(payload), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLatestMergedReleasePullRequestSelectsTagPR(t *testing.T) {
	r := newTestRepo(t, "")
	headSHA := r.git("rev-parse", "HEAD")
	mergedAt := &github.Timestamp{Time: time.Now()}
	pulls := []*github.PullRequest{
		{
			Number:   github.Ptr(1),
			MergedAt: mergedAt,
			Head:     &github.PullRequestBranch{Ref: github.Ptr("feature")},
		},
		{
			Number: github.Ptr(2),
			Head:   &github.PullRequestBranch{Ref: github.Ptr("tagpr-from-v0.1.0")},
			Labels: []*github.Label{{Name: github.Ptr("tagpr")}},
		},
		{
			Number:   github.Ptr(3),
			MergedAt: mergedAt,
			Head:     &github.PullRequestBranch{Ref: github.Ptr("tagpr-from-v0.1.0")},
			Labels:   []*github.Label{{Name: github.Ptr("tagpr")}},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Songmu/tagpr/commits/"+headSHA+"/pulls",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(pulls); err != nil {
				t.Errorf("failed to encode pull requests: %v", err)
			}
		})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	cli := github.NewClient(nil)
	cli.BaseURL = u
	tp := &tagpr{
		c: &commander{
			gitPath:   "git",
			dir:       r.dir,
			outStream: io.Discard,
			errStream: io.Discard,
		},
		gh:    cli,
		owner: "Songmu",
		repo:  "tagpr",
	}

	pr, err := tp.latestMergedReleasePullRequest(context.Background())
	if err != nil {
		t.Fatalf("latestMergedReleasePullRequest() failed: %v", err)
	}
	if pr.GetNumber() != 3 {
		t.Errorf("latestMergedReleasePullRequest() returned #%d, want #3", pr.GetNumber())
	}
}

// newTestTagpr builds a tagpr which talks to a stub GitHub API server. The
// target_commitish passed to the release notes generation API is stored into
// the returned pointer.
func newTestTagpr(t *testing.T, r *testRepo, cfg *config) (*tagpr, *string) {
	t.Helper()
	// the push event payload is opt-in for each test case
	t.Setenv(envGitHubEventName, "")
	t.Setenv(envGitHubEventPath, "")
	var targetCommitish string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Songmu/tagpr/releases/generate-notes",
		func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				TargetCommitish string `json:"target_commitish"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode the request body: %v", err)
			}
			targetCommitish = body.TargetCommitish
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"name": "v0.2.0", "body": "release notes"})
		})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	cli := github.NewClient(nil)
	cli.BaseURL = u

	return &tagpr{
		c: &commander{
			gitPath: "git", outStream: os.Stdout, errStream: os.Stderr, dir: r.dir},
		gh:    cli,
		cfg:   cfg,
		out:   os.Stdout,
		owner: "Songmu",
		repo:  "tagpr",
	}, &targetCommitish
}

func newTestConfig(versionFile string) *config {
	return &config{
		releaseBranch: github.Ptr("main"),
		versionFile:   github.Ptr(versionFile),
		vPrefix:       github.Ptr(true),
		release:       github.Ptr("false"),
	}
}

// TestTagReleaseMergeMethods ensures that tagpr tags the merged HEAD and
// generates the release notes against the exact pre-merge base SHA for all the
// three merge methods GitHub provides. Note that "HEAD~" is a commit of the
// release pull request itself when "Rebase and merge" was used.
func TestTagReleaseMergeMethods(t *testing.T) {
	for _, method := range []string{"merge", "squash", "rebase"} {
		t.Run(method, func(t *testing.T) {
			r := newTestRepo(t, "")
			baseSHA := r.merge(method)
			headSHA := r.git("rev-parse", "HEAD")

			tp, targetCommitish := newTestTagpr(t, r, newTestConfig("-"))
			currVer, err := newSemver("v0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			pr := &github.PullRequest{
				Number: github.Ptr(1),
				Head:   &github.PullRequestBranch{Ref: github.Ptr("tagpr-from-v0.1.0")},
				Base:   &github.PullRequestBranch{SHA: github.Ptr(baseSHA)},
				Labels: []*github.Label{{Name: github.Ptr("tagpr")}},
			}
			if err := tp.tagRelease(context.Background(), pr, currVer, "v0.1.0"); err != nil {
				t.Fatalf("tagRelease() failed: %v", err)
			}

			if got := r.git("rev-parse", "v0.1.1"); got != headSHA {
				t.Errorf("the tag v0.1.1 points to %s, want the merged HEAD %s", got, headSHA)
			}
			if *targetCommitish != baseSHA {
				t.Errorf("target_commitish = %s, want the base SHA %s", *targetCommitish, baseSHA)
			}
			if got := r.git("symbolic-ref", "--short", "HEAD"); got != "main" {
				t.Errorf("the current branch is %s, want main", got)
			}
		})
	}
}

// TestTagReleaseVersionFileDetection ensures that the version file detection,
// which checks out the base commit of the release pull request, works for all
// the merge methods.
func TestTagReleaseVersionFileDetection(t *testing.T) {
	for _, method := range []string{"merge", "squash", "rebase"} {
		t.Run(method, func(t *testing.T) {
			r := newTestRepo(t, "package main\n\nconst version = \"0.1.0\"\n")
			baseSHA := r.merge(method)
			headSHA := r.git("rev-parse", "HEAD")

			// detectVersionFile scans the current working directory
			t.Chdir(r.dir)

			tp, targetCommitish := newTestTagpr(t, r, newTestConfig(""))
			currVer, err := newSemver("v0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			pr := &github.PullRequest{
				Number: github.Ptr(1),
				Head:   &github.PullRequestBranch{Ref: github.Ptr("tagpr-from-v0.1.0")},
				Base:   &github.PullRequestBranch{SHA: github.Ptr(baseSHA)},
			}
			if err := tp.tagRelease(context.Background(), pr, currVer, "v0.1.0"); err != nil {
				t.Fatalf("tagRelease() failed: %v", err)
			}

			// the version file of the merged HEAD holds 0.2.0
			if got := r.git("rev-parse", "v0.2.0"); got != headSHA {
				t.Errorf("the tag v0.2.0 points to %s, want the merged HEAD %s", got, headSHA)
			}
			if *targetCommitish != baseSHA {
				t.Errorf("target_commitish = %s, want the base SHA %s", *targetCommitish, baseSHA)
			}
		})
	}
}

// TestTagReleaseInvalidBaseSHA ensures that tagpr never falls back to a guessed
// commit when the base SHA is unavailable.
func TestTagReleaseInvalidBaseSHA(t *testing.T) {
	tests := map[string]*github.PullRequest{
		"missing base": {Number: github.Ptr(1)},
		"empty base SHA": {
			Number: github.Ptr(1),
			Base:   &github.PullRequestBranch{SHA: github.Ptr("")},
		},
	}
	for name, pr := range tests {
		t.Run(name, func(t *testing.T) {
			r := newTestRepo(t, "")
			r.merge("merge")

			tp, _ := newTestTagpr(t, r, newTestConfig("-"))
			currVer, err := newSemver("v0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			if err := tp.tagRelease(context.Background(), pr, currVer, "v0.1.0"); err == nil {
				t.Error("tagRelease() expected an error, but got nil")
			}
		})
	}
}

func TestPushEventBeforeSHA(t *testing.T) {
	tests := map[string]struct {
		payload string
		want    string
		wantErr bool
	}{
		"before":            {payload: `{"before":"cafebabe","after":"deadbeef"}`, want: "cafebabe"},
		"missing before":    {payload: `{"after":"deadbeef"}`},
		"empty before":      {payload: `{"before":"","after":"deadbeef"}`},
		"all-zero before":   {payload: `{"before":"` + strings.Repeat("0", 40) + `"}`},
		"null before":       {payload: `{"before":null}`},
		"malformed JSON":    {payload: `{"before":`, wantErr: true},
		"trailing garbage":  {payload: `{"before":"cafebabe"} garbage`, wantErr: true},
		"multiple values":   {payload: `{"before":"cafebabe"} {}`, wantErr: true},
		"not a JSON object": {payload: `["before"]`, wantErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := pushEventBeforeSHA(writeEventFile(t, tt.payload))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("pushEventBeforeSHA() expected an error, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("pushEventBeforeSHA() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("pushEventBeforeSHA() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("no path", func(t *testing.T) {
		got, err := pushEventBeforeSHA("")
		if err != nil || got != "" {
			t.Errorf("pushEventBeforeSHA(\"\") = %q, %v; want empty string and no error", got, err)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		got, err := pushEventBeforeSHA(filepath.Join(t.TempDir(), "no-such-file.json"))
		if err != nil || got != "" {
			t.Errorf("pushEventBeforeSHA() = %q, %v; want empty string and no error", got, err)
		}
	})

	t.Run("unreadable path", func(t *testing.T) {
		if _, err := pushEventBeforeSHA(t.TempDir()); err == nil {
			t.Error("pushEventBeforeSHA() expected an error, but got nil")
		}
	})
}

// TestReleaseBoundarySHA ensures that the push event payload takes precedence over
// the base SHA of the release pull request, and that the base SHA is used as a
// fallback only when the payload provides no usable "before".
func TestReleaseBoundarySHA(t *testing.T) {
	basePR := &github.PullRequest{
		Number: github.Ptr(1),
		Base:   &github.PullRequestBranch{SHA: github.Ptr("basesha")},
	}
	tests := map[string]struct {
		eventName string
		payload   *string
		unsetEnv  bool
		pr        *github.PullRequest
		want      string
		wantErr   bool
	}{
		"event takes precedence": {
			eventName: "push",
			payload:   github.Ptr(`{"before":"eventsha"}`),
			pr:        basePR,
			want:      "eventsha",
		},
		"event takes precedence without a pull request": {
			eventName: "push",
			payload:   github.Ptr(`{"before":"eventsha"}`),
			want:      "eventsha",
		},
		"non-push event ignores before": {
			eventName: "pull_request",
			payload:   github.Ptr(`{"before":"unrelatedsha"}`),
			pr:        basePR,
			want:      "basesha",
		},
		"fallback when the env is unset": {
			unsetEnv: true, pr: basePR, want: "basesha"},
		"fallback when before is missing": {
			eventName: "push",
			payload:   github.Ptr(`{"after":"deadbeef"}`),
			pr:        basePR,
			want:      "basesha",
		},
		"fallback when before is empty": {
			eventName: "push",
			payload:   github.Ptr(`{"before":""}`),
			pr:        basePR,
			want:      "basesha",
		},
		"fallback when before is all-zero": {
			eventName: "push",
			payload:   github.Ptr(`{"before":"` + strings.Repeat("0", 40) + `"}`),
			pr:        basePR,
			want:      "basesha",
		},
		"malformed payload": {
			eventName: "push",
			payload:   github.Ptr(`{"before":`),
			pr:        basePR,
			wantErr:   true,
		},
		"fallback without a base": {
			unsetEnv: true, pr: &github.PullRequest{Number: github.Ptr(1)}, wantErr: true},
		"fallback without a pull request": {
			unsetEnv: true, pr: nil, wantErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.unsetEnv {
				// t.Setenv registers the restoration of the original state,
				// then the variable is removed to emulate a non-Actions run
				t.Setenv(envGitHubEventName, "")
				t.Setenv(envGitHubEventPath, "")
				os.Unsetenv(envGitHubEventName)
				os.Unsetenv(envGitHubEventPath)
			} else {
				t.Setenv(envGitHubEventName, tt.eventName)
				t.Setenv(envGitHubEventPath, writeEventFile(t, *tt.payload))
			}
			got, err := releaseBoundarySHA(tt.pr)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("releaseBoundarySHA() expected an error, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("releaseBoundarySHA() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("releaseBoundarySHA() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithCheckoutRestoresBranchOnError(t *testing.T) {
	r := newTestRepo(t, "")
	boundarySHA := r.git("rev-parse", "HEAD")
	r.commit("later.txt", "later\n", "advance main")
	sentinel := errors.New("detection failed")
	tp := &tagpr{
		c: &commander{
			gitPath:   "git",
			dir:       r.dir,
			outStream: io.Discard,
			errStream: io.Discard,
		},
	}

	err := tp.withCheckout(boundarySHA, "main", func() error {
		if got := r.git("rev-parse", "HEAD"); got != boundarySHA {
			t.Errorf("HEAD = %s, want boundary %s", got, boundarySHA)
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("withCheckout() error = %v, want %v", err, sentinel)
	}
	if got := r.git("symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Errorf("current branch = %s, want main", got)
	}
}

// TestTagReleaseStaleBase ensures that the "before" SHA of the push event is used
// even when the base SHA of the release pull request is stale because the release
// branch advanced after the release pull request was last updated.
func TestTagReleaseStaleBase(t *testing.T) {
	for _, method := range []string{"merge", "squash", "rebase"} {
		t.Run(method, func(t *testing.T) {
			r := newTestRepo(t, "")
			staleBaseSHA := r.git("rev-parse", "HEAD")
			beforeSHA := r.advanceMain("other.md", "other\n", "another pull request")
			if staleBaseSHA == beforeSHA {
				t.Fatal("the release branch was not advanced")
			}
			r.merge(method)
			headSHA := r.git("rev-parse", "HEAD")

			tp, targetCommitish := newTestTagpr(t, r, newTestConfig("-"))
			t.Setenv(envGitHubEventName, "push")
			t.Setenv(envGitHubEventPath,
				writeEventFile(t, `{"before":"`+beforeSHA+`","after":"`+headSHA+`"}`))
			currVer, err := newSemver("v0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			pr := &github.PullRequest{
				Number: github.Ptr(1),
				Head:   &github.PullRequestBranch{Ref: github.Ptr("tagpr-from-v0.1.0")},
				Base:   &github.PullRequestBranch{SHA: github.Ptr(staleBaseSHA)},
				Labels: []*github.Label{{Name: github.Ptr("tagpr")}},
			}
			if err := tp.tagRelease(context.Background(), pr, currVer, "v0.1.0"); err != nil {
				t.Fatalf("tagRelease() failed: %v", err)
			}

			if got := r.git("rev-parse", "v0.1.1"); got != headSHA {
				t.Errorf("the tag v0.1.1 points to %s, want the merged HEAD %s", got, headSHA)
			}
			if *targetCommitish != beforeSHA {
				t.Errorf("target_commitish = %s, want the event before SHA %s",
					*targetCommitish, beforeSHA)
			}
		})
	}
}
