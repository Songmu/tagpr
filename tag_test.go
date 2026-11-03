package tagpr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Songmu/gitconfig"
	"github.com/google/go-github/v93/github"
)

func TestMergedBaseSHA(t *testing.T) {
	tests := map[string]struct {
		pr      *github.PullRequest
		want    string
		wantErr bool
	}{
		"valid base SHA": {
			pr: &github.PullRequest{
				Base: &github.PullRequestBranch{SHA: new("deadbeef")},
			},
			want: "deadbeef",
		},
		"nil pull request": {
			pr:      nil,
			wantErr: true,
		},
		"missing base": {
			pr:      &github.PullRequest{Number: new(10)},
			wantErr: true,
		},
		"empty base SHA": {
			pr: &github.PullRequest{
				Number: new(10),
				Base:   &github.PullRequestBranch{SHA: new("")},
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
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
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

func TestLatestMergedReleasePullRequestPaginatesAndMatchesBase(t *testing.T) {
	r := newTestRepo(t, "")
	headSHA := r.git("rev-parse", "HEAD")
	mergedAt := &github.Timestamp{Time: time.Now()}
	firstPage := []*github.PullRequest{
		{
			Number:   new(1),
			MergedAt: mergedAt,
			Head:     &github.PullRequestBranch{Ref: new("feature")},
		},
		{
			Number: new(2),
			Head:   &github.PullRequestBranch{Ref: new("tagpr-from-v0.1.0")},
			Base:   &github.PullRequestBranch{Ref: new("main")},
			Labels: []*github.Label{{Name: "tagpr"}},
		},
		{
			Number:   new(3),
			MergedAt: mergedAt,
			Head:     &github.PullRequestBranch{Ref: new("tagpr-from-v0.1.0")},
			Base:     &github.PullRequestBranch{Ref: new("release/v1")},
			Labels:   []*github.Label{{Name: "tagpr"}},
		},
	}
	secondPage := []*github.PullRequest{
		{
			Number:   new(4),
			MergedAt: mergedAt,
			Head:     &github.PullRequestBranch{Ref: new("tagpr-from-v0.1.0")},
			Base:     &github.PullRequestBranch{Ref: new("main")},
			Labels:   []*github.Label{{Name: "tagpr"}},
		},
	}

	mux := http.NewServeMux()
	var srv *httptest.Server
	var requestedPages []string
	mux.HandleFunc("/repos/Songmu/tagpr/commits/"+headSHA+"/pulls",
		func(w http.ResponseWriter, req *http.Request) {
			page := req.URL.Query().Get("page")
			if page == "" {
				page = "1"
			}
			requestedPages = append(requestedPages, page)
			if req.URL.Query().Get("per_page") != "100" {
				t.Errorf("per_page = %q, want 100", req.URL.Query().Get("per_page"))
			}
			w.Header().Set("Content-Type", "application/json")
			pulls := secondPage
			if page == "1" {
				pulls = firstPage
				w.Header().Set("Link",
					"<"+srv.URL+req.URL.Path+"?page=2&per_page=100>; rel=\"next\"")
			}
			if err := json.NewEncoder(w).Encode(pulls); err != nil {
				t.Errorf("failed to encode pull requests: %v", err)
			}
		})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cli, err := github.NewClient(github.WithURLs(new(srv.URL+"/"), nil))
	if err != nil {
		t.Fatal(err)
	}
	tp := &tagpr{
		c: &commander{
			gitPath:   "git",
			dir:       r.dir,
			outStream: io.Discard,
			errStream: io.Discard,
		},
		gh:    cli,
		cfg:   &config{releaseBranch: new("main")},
		owner: "Songmu",
		repo:  "tagpr",
	}

	pr, err := tp.latestMergedReleasePullRequest(context.Background())
	if err != nil {
		t.Fatalf("latestMergedReleasePullRequest() failed: %v", err)
	}
	if pr.GetNumber() != 4 {
		t.Errorf("latestMergedReleasePullRequest() returned #%d, want #4", pr.GetNumber())
	}
	if got := strings.Join(requestedPages, ","); got != "1,2" {
		t.Errorf("requested pages = %s, want 1,2", got)
	}
}

type testReleaseRequests struct {
	targetCommitish    string
	generateNotesCalls int
	createReleaseCalls int
}

// newTestTagpr builds a tagpr which talks to a stub GitHub API server.
func newTestTagpr(t *testing.T, r *testRepo, cfg *config) (*tagpr, *testReleaseRequests) {
	t.Helper()
	// the push event payload is opt-in for each test case
	t.Setenv(envGitHubEventName, "")
	t.Setenv(envGitHubEventPath, "")
	requests := &testReleaseRequests{}
	mergeCommitSHA := r.git("rev-parse", "HEAD")
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Songmu/tagpr/releases/generate-notes",
		func(w http.ResponseWriter, req *http.Request) {
			requests.generateNotesCalls++
			var body struct {
				TargetCommitish string `json:"target_commitish"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode the request body: %v", err)
			}
			requests.targetCommitish = body.TargetCommitish
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"name": "v0.2.0", "body": "release notes"})
		})
	mux.HandleFunc("/repos/Songmu/tagpr/releases",
		func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if req.Method == http.MethodGet {
				json.NewEncoder(w).Encode([]any{})
				return
			}
			requests.createReleaseCalls++
			json.NewEncoder(w).Encode(map[string]string{
				"name": "v0.2.0", "body": "release notes"})
		})
	mux.HandleFunc("/repos/Songmu/tagpr/commits/", func(w http.ResponseWriter, req *http.Request) {
		if !strings.HasSuffix(req.URL.Path, "/pulls") {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{
			"number":           1,
			"merged_at":        "2026-09-22T00:00:00Z",
			"merge_commit_sha": mergeCommitSHA,
			"head": map[string]string{
				"ref": "tagpr-from-v0.1.0",
			},
			"base": map[string]string{
				"ref": "main",
			},
			"labels": []map[string]string{{"name": "tagpr"}},
		}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cli, err := github.NewClient(github.WithURLs(new(srv.URL+"/"), nil))
	if err != nil {
		t.Fatal(err)
	}

	return &tagpr{
		c: &commander{
			gitPath: "git", outStream: os.Stdout, errStream: os.Stderr, dir: r.dir},
		gh:    cli,
		cfg:   cfg,
		out:   os.Stdout,
		owner: "Songmu",
		repo:  "tagpr",
	}, requests
}

func newTestConfig(versionFile string) *config {
	return &config{
		releaseBranch: new("main"),
		versionFile:   new(versionFile),
		vPrefix:       new(true),
		release:       new("false"),
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

			cfg := newTestConfig("-")
			releaseMode := "true"
			if method == "squash" {
				releaseMode = "draft"
			}
			cfg.release = &releaseMode
			tp, requests := newTestTagpr(t, r, cfg)
			currVer, err := newSemver("v0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			pr := &github.PullRequest{
				Number: new(1),
				Head:   &github.PullRequestBranch{Ref: new("tagpr-from-v0.1.0")},
				Base:   &github.PullRequestBranch{SHA: new(baseSHA)},
				Labels: []*github.Label{{Name: "tagpr"}},
			}
			if err := tp.tagRelease(context.Background(), pr, currVer, "v0.1.0"); err != nil {
				t.Fatalf("tagRelease() failed: %v", err)
			}

			if got := r.git("rev-parse", "v0.1.1"); got != headSHA {
				t.Errorf("the tag v0.1.1 points to %s, want the merged HEAD %s", got, headSHA)
			}
			if requests.targetCommitish != baseSHA {
				t.Errorf("target_commitish = %s, want the base SHA %s",
					requests.targetCommitish, baseSHA)
			}
			if requests.generateNotesCalls != 1 || requests.createReleaseCalls != 1 {
				t.Errorf("release API calls = generate:%d create:%d, want 1 each",
					requests.generateNotesCalls, requests.createReleaseCalls)
			}
			if got := r.git("symbolic-ref", "--short", "HEAD"); got != "main" {
				t.Errorf("the current branch is %s, want main", got)
			}
		})
	}
}

func TestTagSigningEnabled(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)

	tests := map[string]struct {
		value   string
		want    bool
		wantErr bool
	}{
		"unset":   {},
		"false":   {value: "false"},
		"true":    {value: "true", want: true},
		"invalid": {value: "invalid", wantErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cmd := exec.Command("git", "init", "--quiet", dir)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git init failed: %v\n%s", err, out)
			}
			if tt.value != "" {
				cmd := exec.Command("git", "-C", dir, "config", "tag.gpgSign", tt.value)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git config failed: %v\n%s", err, out)
				}
			}

			tp := &tagpr{c: &commander{
				gitPath: "git", dir: dir, outStream: io.Discard, errStream: io.Discard,
			}}
			got, err := tp.tagSigningEnabled()
			if tt.wantErr {
				if err == nil {
					t.Fatal("tagSigningEnabled() expected an error, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("tagSigningEnabled() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("tagSigningEnabled() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestTagReleaseSignedTag(t *testing.T) {
	sshKeygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen is required to test signed tags")
	}

	r := newTestRepo(t, "")
	baseSHA := r.merge("merge")
	headSHA := r.git("rev-parse", "HEAD")

	signingKey := filepath.Join(t.TempDir(), "signing-key")
	cmd := exec.Command(sshKeygen, "-q", "-t", "ed25519", "-N", "", "-f", signingKey)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen failed: %v\n%s", err, out)
	}
	r.git("config", "gpg.format", "ssh")
	r.git("config", "user.signingKey", signingKey)
	r.git("config", "tag.gpgSign", "true")

	tp, _ := newTestTagpr(t, r, newTestConfig("-"))
	currVer, err := newSemver("v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	pr := &github.PullRequest{
		Number: new(1),
		Base:   &github.PullRequestBranch{SHA: new(baseSHA)},
		Labels: []*github.Label{{Name: "tagpr"}},
	}
	if err := tp.tagRelease(context.Background(), pr, currVer, "v0.1.0"); err != nil {
		t.Fatalf("tagRelease() failed: %v", err)
	}

	if got := r.git("cat-file", "-t", "v0.1.1"); got != "tag" {
		t.Errorf("tag type = %q, want annotated tag", got)
	}
	if got := r.git("rev-parse", "v0.1.1^{}"); got != headSHA {
		t.Errorf("the tag v0.1.1 points to %s, want the merged HEAD %s", got, headSHA)
	}
	tagObject := r.git("cat-file", "tag", "v0.1.1")
	if !strings.Contains(tagObject, "\nRelease v0.1.1\n") {
		t.Errorf("tag object does not contain the release message:\n%s", tagObject)
	}
	if !strings.Contains(tagObject, "-----BEGIN SSH SIGNATURE-----") {
		t.Errorf("tag object does not contain an SSH signature:\n%s", tagObject)
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

			tp, requests := newTestTagpr(t, r, newTestConfig(""))
			currVer, err := newSemver("v0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			pr := &github.PullRequest{
				Number: new(1),
				Head:   &github.PullRequestBranch{Ref: new("tagpr-from-v0.1.0")},
				Base:   &github.PullRequestBranch{SHA: new(baseSHA)},
			}
			if err := tp.tagRelease(context.Background(), pr, currVer, "v0.1.0"); err != nil {
				t.Fatalf("tagRelease() failed: %v", err)
			}

			// the version file of the merged HEAD holds 0.2.0
			if got := r.git("rev-parse", "v0.2.0"); got != headSHA {
				t.Errorf("the tag v0.2.0 points to %s, want the merged HEAD %s", got, headSHA)
			}
			if requests.generateNotesCalls != 0 || requests.createReleaseCalls != 0 {
				t.Errorf("release API calls = generate:%d create:%d, want 0 each",
					requests.generateNotesCalls, requests.createReleaseCalls)
			}
		})
	}
}

// TestTagReleaseInvalidBaseSHA ensures that tagpr never falls back to a guessed
// commit when the base SHA is unavailable.
func TestTagReleaseInvalidBaseSHA(t *testing.T) {
	tests := map[string]*github.PullRequest{
		"missing base": {Number: new(1)},
		"empty base SHA": {
			Number: new(1),
			Base:   &github.PullRequestBranch{SHA: new("")},
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
	const (
		releaseRef = "refs/heads/main"
		headSHA    = "deadbeef"
	)
	tests := map[string]struct {
		payload string
		want    string
		wantErr bool
	}{
		"before": {payload: `{"before":"cafebabe","after":"deadbeef",` +
			`"ref":"refs/heads/main"}`, want: "cafebabe"},
		"wrong ref": {payload: `{"before":"cafebabe","after":"deadbeef",` +
			`"ref":"refs/heads/feature"}`},
		"wrong after": {payload: `{"before":"cafebabe","after":"other",` +
			`"ref":"refs/heads/main"}`},
		"missing before": {payload: `{"after":"deadbeef","ref":"refs/heads/main"}`},
		"empty before": {payload: `{"before":"","after":"deadbeef",` +
			`"ref":"refs/heads/main"}`},
		"all-zero before": {payload: `{"before":"` + strings.Repeat("0", 40) +
			`","after":"deadbeef","ref":"refs/heads/main"}`},
		"null before": {payload: `{"before":null,"after":"deadbeef",` +
			`"ref":"refs/heads/main"}`},
		"malformed JSON":    {payload: `{"before":`, wantErr: true},
		"trailing garbage":  {payload: `{"before":"cafebabe"} garbage`, wantErr: true},
		"multiple values":   {payload: `{"before":"cafebabe"} {}`, wantErr: true},
		"not a JSON object": {payload: `["before"]`, wantErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := pushEventBeforeSHA(
				writeEventFile(t, tt.payload), releaseRef, headSHA)
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
		got, err := pushEventBeforeSHA("", releaseRef, headSHA)
		if err != nil || got != "" {
			t.Errorf("pushEventBeforeSHA(\"\") = %q, %v; want empty string and no error", got, err)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		got, err := pushEventBeforeSHA(
			filepath.Join(t.TempDir(), "no-such-file.json"), releaseRef, headSHA)
		if err != nil || got != "" {
			t.Errorf("pushEventBeforeSHA() = %q, %v; want empty string and no error", got, err)
		}
	})

	t.Run("unreadable path", func(t *testing.T) {
		if _, err := pushEventBeforeSHA(t.TempDir(), releaseRef, headSHA); err == nil {
			t.Error("pushEventBeforeSHA() expected an error, but got nil")
		}
	})
}

// TestReleaseBoundarySHA ensures that the push event payload takes precedence over
// the base SHA of the release pull request, and that the base SHA is used as a
// fallback only when the payload provides no usable "before".
func TestReleaseBoundarySHA(t *testing.T) {
	r := newTestRepo(t, "")
	headSHA := r.git("rev-parse", "HEAD")
	tp := &tagpr{
		c: &commander{
			gitPath:   "git",
			dir:       r.dir,
			outStream: io.Discard,
			errStream: io.Discard,
		},
		cfg: &config{releaseBranch: new("main")},
	}
	basePR := &github.PullRequest{
		Number: new(1),
		Base:   &github.PullRequestBranch{SHA: new("basesha")},
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
			payload: new(`{"before":"eventsha","after":"` + headSHA +
				`","ref":"refs/heads/main"}`),

			pr:   basePR,
			want: "eventsha",
		},
		"event takes precedence without a pull request": {
			eventName: "push",
			payload: new(`{"before":"eventsha","after":"` + headSHA +
				`","ref":"refs/heads/main"}`),

			want: "eventsha",
		},
		"wrong ref falls back": {
			eventName: "push",
			payload: new(`{"before":"eventsha","after":"` + headSHA +
				`","ref":"refs/heads/feature"}`),

			pr:   basePR,
			want: "basesha",
		},
		"wrong after falls back": {
			eventName: "push",
			payload: new(`{"before":"eventsha","after":"other",` +
				`"ref":"refs/heads/main"}`),

			pr:   basePR,
			want: "basesha",
		},
		"non-push event ignores before": {
			eventName: "pull_request",
			payload:   new(`{"before":"unrelatedsha"}`),
			pr:        basePR,
			want:      "basesha",
		},
		"fallback when the env is unset": {
			unsetEnv: true, pr: basePR, want: "basesha"},
		"fallback when before is missing": {
			eventName: "push",
			payload: new(`{"after":"` + headSHA +
				`","ref":"refs/heads/main"}`),

			pr:   basePR,
			want: "basesha",
		},
		"fallback when before is empty": {
			eventName: "push",
			payload: new(`{"before":"","after":"` + headSHA +
				`","ref":"refs/heads/main"}`),

			pr:   basePR,
			want: "basesha",
		},
		"fallback when before is all-zero": {
			eventName: "push",
			payload: new(`{"before":"` + strings.Repeat("0", 40) +
				`","after":"` + headSHA + `","ref":"refs/heads/main"}`),

			pr:   basePR,
			want: "basesha",
		},
		"malformed payload": {
			eventName: "push",
			payload:   new(`{"before":`),
			pr:        basePR,
			wantErr:   true,
		},
		"fallback without a base": {
			unsetEnv: true, pr: &github.PullRequest{Number: new(1)}, wantErr: true},
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
			got, err := tp.releaseBoundarySHA(tt.pr)
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

			cfg := newTestConfig("-")
			cfg.release = new("true")
			tp, requests := newTestTagpr(t, r, cfg)
			t.Setenv(envGitHubEventName, "push")
			t.Setenv(envGitHubEventPath,
				writeEventFile(t, `{"before":"`+beforeSHA+`","after":"`+headSHA+
					`","ref":"refs/heads/main"}`))
			currVer, err := newSemver("v0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			pr := &github.PullRequest{
				Number: new(1),
				Head:   &github.PullRequestBranch{Ref: new("tagpr-from-v0.1.0")},
				Base:   &github.PullRequestBranch{SHA: new(staleBaseSHA)},
				Labels: []*github.Label{{Name: "tagpr"}},
			}
			if err := tp.tagRelease(context.Background(), pr, currVer, "v0.1.0"); err != nil {
				t.Fatalf("tagRelease() failed: %v", err)
			}

			if got := r.git("rev-parse", "v0.1.1"); got != headSHA {
				t.Errorf("the tag v0.1.1 points to %s, want the merged HEAD %s", got, headSHA)
			}
			if requests.targetCommitish != beforeSHA {
				t.Errorf("target_commitish = %s, want the event before SHA %s",
					requests.targetCommitish, beforeSHA)
			}
		})
	}
}

func TestPlanModeOutputsReleasePlan(t *testing.T) {
	r := newTestRepo(t, "")
	t.Chdir(r.dir)
	baseSHA := r.merge("merge")
	targetSHA := r.git("rev-parse", "HEAD")

	tp, _ := newTestTagpr(t, r, newTestConfig("-"))
	tp.runOptions = runOptions{Mode: executionModePlan}
	outputPath := filepath.Join(t.TempDir(), "output")
	t.Setenv("GITHUB_OUTPUT", outputPath)
	t.Setenv(envGitHubEventName, "push")
	t.Setenv(envGitHubEventPath,
		writeEventFile(t, `{"before":"`+baseSHA+`","after":"`+targetSHA+
			`","ref":"refs/heads/main"}`))
	if err := tp.Run(context.Background()); err != nil {
		t.Fatalf("Run() in plan mode failed: %v", err)
	}
	b, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string]string{}
	for line := range strings.SplitSeq(strings.TrimSpace(string(b)), "\n") {
		name, value, _ := strings.Cut(line, "=")
		outputs[name] = value
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(outputs["plan"]), &got); err != nil {
		t.Fatalf("failed to decode plan output: %v", err)
	}
	want := map[string]any{
		"tag":                        "v0.1.1",
		"commit_sha":                 targetSHA,
		"base_tag":                   "v0.1.0",
		"pull_request_number":        float64(1),
		"release_notes_boundary_sha": baseSHA,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("plan = %#v, want %#v", got, want)
	}
	if len(outputs) != 3 || outputs["base_tag"] != "v0.1.0" || outputs["pull_request"] == "" {
		t.Errorf("outputs = %#v, want only plan, base_tag, and pull_request", outputs)
	}
	if got := r.git("tag", "-l", "v0.1.1"); got != "" {
		t.Errorf("plan mode created tag %q", got)
	}
}

func TestApplyReleasePlanAfterBranchAdvances(t *testing.T) {
	r := newTestRepo(t, "")
	t.Chdir(r.dir)
	baseSHA := r.merge("merge")
	targetSHA := r.git("rev-parse", "HEAD")
	r.git("push", "origin", "main")

	tp, _ := newTestTagpr(t, r, newTestConfig("-"))
	plan := releasePlan{
		Tag:                     "v0.1.1",
		CommitSHA:               targetSHA,
		ReleaseNotesBoundarySHA: baseSHA,
		PullRequestNumber:       1,
		BaseTag:                 "v0.1.0",
	}
	tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}

	advancedSHA := r.advanceMain("later.md", "later\n", "advance after release merge")
	r.git("push", "origin", "main")
	r.git("tag", "v0.1.1", targetSHA)
	if err := tp.Run(context.Background()); err != nil {
		t.Fatalf("Run() in apply mode failed: %v", err)
	}
	if got := r.git("rev-parse", "v0.1.1"); got != targetSHA {
		t.Errorf("tag points to %s, want %s", got, targetSHA)
	}
	if got := r.git("rev-parse", "origin/main"); got != advancedSHA {
		t.Errorf("origin/main = %s, want %s", got, advancedSHA)
	}
	if got := strings.Fields(r.git(
		"ls-remote", "--tags", "origin", "refs/tags/v0.1.1"))[0]; got != targetSHA {
		t.Errorf("remote tag points to %s, want %s", got, targetSHA)
	}

	if err := tp.Run(context.Background()); err != nil {
		t.Fatalf("idempotent Run() in apply mode failed: %v", err)
	}
}

func TestApplyReleasePlanRejectsInterveningReleaseWithLocalPlannedTag(t *testing.T) {
	for _, calendar := range []bool{false, true} {
		t.Run(fmt.Sprintf("calendar=%t", calendar), func(t *testing.T) {
			versionFile, versionContent := "version.go", `const version = "0.1.0"`
			if calendar {
				versionFile, versionContent = "-", ""
			}
			r := newTestRepo(t, versionContent)
			t.Chdir(r.dir)
			baseTag, plannedTag, interveningTag := "v0.1.0", "v0.2.0", "v0.1.1"
			cfg := newTestConfig(versionFile)
			if calendar {
				cfg.calendarVersioning = new("true")
				baseTag, plannedTag, interveningTag = "v2026.1101.0", "v2026.1102.1", "v2026.1102.0"
				r.git("tag", baseTag)
			}
			baseSHA := r.merge("merge")
			targetSHA := r.git("rev-parse", "HEAD")
			r.git("tag", plannedTag, targetSHA)
			r.git("push", "origin", "main")
			tp, _ := newTestTagpr(t, r, cfg)
			r.advanceMain("later.md", "later\n", "intervening release")
			r.git("tag", interveningTag)
			r.git("push", "origin", "main", "refs/tags/"+interveningTag)
			plan := releasePlan{
				Tag:                     plannedTag,
				CommitSHA:               targetSHA,
				BaseTag:                 baseTag,
				PullRequestNumber:       1,
				ReleaseNotesBoundarySHA: baseSHA,
			}
			tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}
			err := tp.Run(context.Background())
			if err == nil || !strings.Contains(err.Error(), "base tag changed") {
				t.Fatalf("Run() error = %v, want a changed base tag error", err)
			}
			if got := r.git("ls-remote", "--tags", "origin", "refs/tags/"+plannedTag); got != "" {
				t.Errorf("rejected plan published tag: %s", got)
			}
		})
	}
}

func TestApplyReleasePlanRejectsNonAncestor(t *testing.T) {
	r := newTestRepo(t, "")
	baseSHA := r.git("rev-parse", "HEAD")
	r.git("checkout", "-b", "unrelated")
	r.commit("unrelated.md", "unrelated\n", "unrelated release")
	targetSHA := r.git("rev-parse", "HEAD")
	r.git("checkout", "main")

	tp, _ := newTestTagpr(t, r, newTestConfig("-"))
	plan := releasePlan{
		Tag:                     "v0.1.1",
		CommitSHA:               targetSHA,
		ReleaseNotesBoundarySHA: baseSHA,
		PullRequestNumber:       1,
		BaseTag:                 "v0.1.0",
	}

	if err := tp.applyReleasePlan(context.Background(), plan); err == nil {
		t.Fatal("applyReleasePlan() expected an error")
	}
}

func TestApplyReleasePlanRejectsUnfetchedReleaseTag(t *testing.T) {
	r := newTestRepo(t, "")
	t.Chdir(r.dir)
	baseSHA := r.merge("merge")
	targetSHA := r.git("rev-parse", "HEAD")
	r.git("push", "origin", "main")
	tp, _ := newTestTagpr(t, r, newTestConfig("-"))

	publisher := &testRepo{dir: t.TempDir(), t: t}
	r.git("clone", r.git("remote", "get-url", "origin"), publisher.dir)
	publisher.git("config", "user.name", "Test")
	publisher.git("config", "user.email", "test@example.com")
	publisher.git("config", "commit.gpgsign", "false")
	publisher.git("config", "tag.gpgsign", "false")
	publisher.git("checkout", "-b", "maintenance", baseSHA)
	publisher.commit("hotfix.md", "fix\n", "maintenance release")
	publisher.git("tag", "v0.1.2")
	publisher.git("push", "origin", "refs/tags/v0.1.2")

	plan := releasePlan{
		Tag:                     "v0.1.1",
		CommitSHA:               targetSHA,
		BaseTag:                 "v0.1.0",
		PullRequestNumber:       1,
		ReleaseNotesBoundarySHA: baseSHA,
	}
	tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}
	err := tp.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "base tag changed") {
		t.Fatalf("Run() error = %v, want a remote base-tag change", err)
	}
	if got := r.git("ls-remote", "--tags", "origin", "refs/tags/v0.1.1"); got != "" {
		t.Errorf("stale plan published a tag: %s", got)
	}
}

func TestApplyReleasePlanFetchesMissingTargetCommit(t *testing.T) {
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(globalConfig, []byte(
		"[user]\nname = Test\nemail = test@example.com\n[tag]\ngpgsign = true\n[gpg]\nprogram = false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	tests := []struct {
		baseTag, tag string
		calendar     bool
	}{
		{baseTag: "v0.1.0", tag: "v0.1.1"},
		{baseTag: "0.1.0", tag: "0.1.1"},
		{baseTag: "v2000.1031.0", tag: "v2000.1101.0", calendar: true},
		{baseTag: "2000.1031.0", tag: "2000.1101.0", calendar: true},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			r := newTestRepo(t, "")
			initialSHA := r.git("rev-parse", "HEAD")
			r.git("tag", "-d", "v0.1.0")
			r.git("tag", tt.baseTag, initialSHA)
			checkout := &testRepo{dir: t.TempDir(), t: t}
			r.git("clone", "--branch", "main", r.git("remote", "get-url", "origin"), checkout.dir)
			checkout.git("config", "tag.gpgsign", "false")
			baseSHA := r.merge("merge")
			targetSHA := r.git("rev-parse", "HEAD")
			r.git("push", "origin", "main", "refs/tags/"+tt.baseTag)
			t.Chdir(checkout.dir)
			cfg := newTestConfig("-")
			cfg.vPrefix = nil
			cfg.conf = filepath.Join(checkout.dir, ".tagpr")
			cfg.gitconfig = &gitconfig.Config{GitPath: "git", File: cfg.conf}
			if tt.calendar {
				cfg.calendarVersioning = new("true")
			}
			tp, _ := newTestTagpr(t, r, cfg)
			tp.c.dir = checkout.dir
			plan := releasePlan{
				Tag:                     tt.tag,
				CommitSHA:               targetSHA,
				BaseTag:                 tt.baseTag,
				PullRequestNumber:       1,
				ReleaseNotesBoundarySHA: baseSHA,
			}
			tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}
			if err := tp.Run(context.Background()); err != nil {
				t.Fatalf("Run() with a target missing from the initial checkout failed: %v", err)
			}
			if got := checkout.git("rev-parse", "refs/tags/"+tt.tag+"^{commit}"); got != targetSHA {
				t.Errorf("tagged %s, want the fetched plan commit %s", got, targetSHA)
			}
		})
	}
}

func TestApplyReleasePlanAcceptsDifferentTagObjectsForSameCommit(t *testing.T) {
	for _, calendar := range []bool{false, true} {
		t.Run(fmt.Sprintf("calendar=%t", calendar), func(t *testing.T) {
			r := newTestRepo(t, "")
			t.Chdir(r.dir)
			baseSHA := r.merge("merge")
			targetSHA := r.git("rev-parse", "HEAD")
			baseTag, tag := "v0.1.0", "v0.1.1"
			cfg := newTestConfig("-")
			if calendar {
				cfg.calendarVersioning = new("true")
				baseTag, tag = "v2000.1031.0", "v2000.1101.0"
				r.git("tag", baseTag, baseSHA)
			}
			r.git("tag", tag, targetSHA)
			r.git("push", "origin", "main", "refs/tags/"+baseTag)
			publisher := &testRepo{dir: t.TempDir(), t: t}
			r.git("clone", "--branch", "main", r.git("remote", "get-url", "origin"), publisher.dir)
			publisher.git("config", "user.name", "Test")
			publisher.git("config", "user.email", "test@example.com")
			publisher.git("config", "tag.gpgsign", "false")
			publisher.git("tag", "-a", "-m", "remote annotation", tag, targetSHA)
			publisher.git("push", "origin", "refs/tags/"+tag)

			tp, _ := newTestTagpr(t, r, cfg)
			plan := releasePlan{
				Tag:                     tag,
				CommitSHA:               targetSHA,
				BaseTag:                 baseTag,
				PullRequestNumber:       1,
				ReleaseNotesBoundarySHA: baseSHA,
			}
			tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}
			if err := tp.Run(context.Background()); err != nil {
				t.Fatalf("Run() with matching peeled commits failed: %v", err)
			}
			if got := r.git("cat-file", "-t", "refs/tags/"+tag); got != "commit" {
				t.Errorf("fetch changed the lightweight local tag to %q", got)
			}
		})
	}
}

func TestReleasePlanReadsCommittedVersion(t *testing.T) {
	for _, versionFile := range []string{"version.go", ""} {
		t.Run("versionFile="+versionFile, func(t *testing.T) {
			r := newTestRepo(t, `const version = "0.1.0"`)
			t.Chdir(r.dir)
			baseSHA := r.merge("merge")
			targetSHA := r.git("rev-parse", "HEAD")
			r.git("push", "origin", "main")
			staged := `const version = "0.9.0"`
			dirty := `const version = "0.9.1"`
			if err := os.WriteFile(filepath.Join(r.dir, "version.go"), []byte(staged), 0644); err != nil {
				t.Fatal(err)
			}
			r.git("add", "version.go")
			if err := os.WriteFile(filepath.Join(r.dir, "version.go"), []byte(dirty), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(r.dir, "aaa.go"), []byte(`const version = "0.1.0"`), 0644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GITHUB_OUTPUT", "")
			if err := os.Unsetenv("GITHUB_OUTPUT"); err != nil {
				t.Fatal(err)
			}
			tp, _ := newTestTagpr(t, r, newTestConfig(versionFile))
			t.Setenv(envGitHubEventName, "push")
			t.Setenv(envGitHubEventPath, writeEventFile(t,
				`{"before":"`+baseSHA+`","after":"`+targetSHA+`","ref":"refs/heads/main"}`))
			var output, diagnostics bytes.Buffer
			tp.out, tp.c.outStream = &output, &diagnostics
			tp.runOptions = runOptions{Mode: executionModePlan}
			if err := tp.Run(context.Background()); err != nil {
				t.Errorf("plan with local version edits failed: %v", err)
			} else {
				var plan releasePlan
				if err := json.Unmarshal(output.Bytes(), &plan); err != nil {
					t.Fatal(err)
				}
				if plan.Tag != "v0.2.0" || plan.CommitSHA != targetSHA {
					t.Errorf("plan used local edits: %#v", plan)
				}
			}
			plan := releasePlan{
				Tag: "v0.9.1", CommitSHA: targetSHA, BaseTag: "v0.1.0",
				PullRequestNumber: 1, ReleaseNotesBoundarySHA: baseSHA,
			}
			tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}
			if err := tp.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "planned tag changed") {
				t.Fatalf("locally edited version error = %v, want a changed-tag error", err)
			}
			plan.Tag = "v0.2.0"
			if err := tp.Run(context.Background()); err != nil {
				t.Fatalf("apply of the committed version failed: %v", err)
			}
			if got := r.git("rev-parse", plan.Tag); got != targetSHA {
				t.Errorf("tag = %s, want %s", got, targetSHA)
			}
			if got := r.git("show", ":version.go"); got != staged {
				t.Errorf("staged version changed: %q", got)
			}
			if got, err := os.ReadFile(filepath.Join(r.dir, "version.go")); err != nil || string(got) != dirty {
				t.Errorf("working version changed: %q, err=%v", got, err)
			}
			if got := r.git("symbolic-ref", "--short", "HEAD"); got != "main" {
				t.Errorf("branch changed to %s", got)
			}
			if strings.Contains(diagnostics.String(), "const version") {
				t.Error("version file contents leaked into diagnostics")
			}
		})
	}
}

func TestReleasePlanRejectsUntrackedVersionFile(t *testing.T) {
	r := newTestRepo(t, "")
	t.Chdir(r.dir)
	baseSHA := r.merge("merge")
	targetSHA := r.git("rev-parse", "HEAD")
	r.git("push", "origin", "main")
	if err := os.WriteFile(filepath.Join(r.dir, "untracked.go"), []byte(`const version = "0.2.0"`), 0644); err != nil {
		t.Fatal(err)
	}
	tp, _ := newTestTagpr(t, r, newTestConfig("untracked.go"))
	t.Setenv(envGitHubEventName, "push")
	t.Setenv(envGitHubEventPath, writeEventFile(t,
		`{"before":"`+baseSHA+`","after":"`+targetSHA+`","ref":"refs/heads/main"}`))
	tp.runOptions = runOptions{Mode: executionModePlan}
	if err := tp.Run(context.Background()); err == nil {
		t.Error("plan accepted a version file absent from the target commit")
	}
	plan := releasePlan{
		Tag: "v0.2.0", CommitSHA: targetSHA, BaseTag: "v0.1.0",
		PullRequestNumber: 1, ReleaseNotesBoundarySHA: baseSHA,
	}
	tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}
	if err := tp.Run(context.Background()); err == nil {
		t.Fatal("apply accepted a version file absent from the target commit")
	}
}

func TestReleasePlanConfiguredVersionFilePaths(t *testing.T) {
	tests := []struct {
		path, tag string
	}{
		{path: "version.go", tag: "v0.3.0"},
		{path: "./version.go", tag: "v0.3.0"},
		{path: "../version.go", tag: "v0.2.0"},
		{path: "absolute", tag: "v0.3.0"},
		{path: "version[1].go", tag: "v0.4.0"},
		{path: "introduced.go", tag: "v0.6.0"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := newTestRepo(t, `const version = "0.1.0"`)
			if err := os.Mkdir(filepath.Join(r.dir, "tools"), 0755); err != nil {
				t.Fatal(err)
			}
			r.commit("tools/version.go", `const version = "0.3.0"`, "module version")
			r.commit("tools/version[1].go", `const version = "0.4.0"`, "literal version path")
			r.commit("tools/version1.go", `const version = "0.5.0"`, "other version path")
			baseSHA := r.merge("merge")
			if tt.path == "introduced.go" {
				r.commit("tools/introduced.go", `const version = "0.6.0"`, "introduce version file")
			}
			targetSHA := r.git("rev-parse", "HEAD")
			r.git("push", "origin", "main")
			workdir := filepath.Join(r.dir, "tools")
			t.Chdir(workdir)
			path := tt.path
			if path == "absolute" {
				path = filepath.Join(workdir, "version.go")
			}
			tp, _ := newTestTagpr(t, r, newTestConfig(path))
			tp.c.dir = workdir
			t.Setenv(envGitHubEventName, "push")
			t.Setenv(envGitHubEventPath, writeEventFile(t,
				`{"before":"`+baseSHA+`","after":"`+targetSHA+`","ref":"refs/heads/main"}`))
			t.Setenv("GITHUB_OUTPUT", "")
			if err := os.Unsetenv("GITHUB_OUTPUT"); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			tp.out = &output
			tp.runOptions = runOptions{Mode: executionModePlan}
			if err := tp.Run(context.Background()); err != nil {
				t.Fatalf("plan from a subdirectory failed: %v", err)
			}
			var plan releasePlan
			if err := json.Unmarshal(output.Bytes(), &plan); err != nil {
				t.Fatal(err)
			}
			if plan.Tag != tt.tag {
				t.Errorf("plan tag = %s, want %s", plan.Tag, tt.tag)
			}
			plan.Tag = tt.tag
			tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}
			if err := tp.Run(context.Background()); err != nil {
				t.Fatalf("apply from a subdirectory failed: %v", err)
			}
			if got := r.git("rev-parse", plan.Tag); got != targetSHA {
				t.Errorf("tag = %s, want %s", got, targetSHA)
			}
		})
	}
}

func TestAutoModeRetriesWithOriginalReleaseBase(t *testing.T) {
	tests := []struct {
		name, versionFile, content, prefix, baseTag, tag string
		calendar, tagged                                 bool
		aliases                                          []string
	}{
		{name: "no version file", versionFile: "-", baseTag: "v0.1.0", tag: "v0.1.1"},
		{name: "version file", versionFile: "version.go", content: `const version = "0.1.0"`,
			baseTag: "v0.1.0", tag: "v0.2.0"},
		{name: "first release", versionFile: "-", tag: "v0.0.1"},
		{name: "namespace", versionFile: "-", prefix: "tools",
			baseTag: "tools/v0.1.0", tag: "tools/v0.1.1"},
		{name: "CalVer retry on a later date", versionFile: "-", calendar: true, tagged: true,
			baseTag: "v2000.1031.0", tag: "v2000.1101.0"},
		{name: "floating aliases", versionFile: "-", baseTag: "v1.0.0", tag: "v1.0.1",
			aliases: []string{"v1", "v1.0"}},
		{name: "floating aliases without v", versionFile: "-", baseTag: "1.0.0", tag: "1.0.1",
			aliases: []string{"1", "1.0"}},
		{name: "namespaced floating aliases", versionFile: "-", prefix: "tools",
			baseTag: "tools/v1.0.0", tag: "tools/v1.0.1", aliases: []string{"tools/v1", "tools/v1.0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRepo(t, tt.content)
			t.Chdir(r.dir)
			baseSHA := r.merge("merge")
			targetSHA := r.git("rev-parse", "HEAD")
			if tt.baseTag != "v0.1.0" {
				r.git("tag", "-d", "v0.1.0")
				if tt.baseTag != "" {
					r.git("tag", tt.baseTag, baseSHA)
				}
			}
			if tt.tagged {
				r.git("tag", tt.tag, targetSHA)
				r.git("push", "origin", "refs/tags/"+tt.tag)
			}
			for _, alias := range tt.aliases {
				r.git("tag", alias, baseSHA)
			}
			prefix := normalizeTagPrefix(tt.prefix)
			baseVersion := strings.TrimPrefix(tt.baseTag, prefix)
			if baseVersion == "" {
				baseVersion = "v0.0.0"
			}
			var notesTags, notesBases []string
			var createCalls int
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/Songmu/tagpr/commits/"+targetSHA+"/pulls",
				func(w http.ResponseWriter, req *http.Request) {
					json.NewEncoder(w).Encode([]map[string]any{{
						"number": 1, "merged_at": "2026-11-02T00:00:00Z",
						"merge_commit_sha": targetSHA,
						"head": map[string]string{
							"ref": branchPrefix + branchSafePrefix(prefix) + baseVersion,
						},
						"base":   map[string]string{"ref": "main", "sha": baseSHA},
						"labels": []map[string]string{{"name": "tagpr"}},
					}})
				})
			mux.HandleFunc("/repos/Songmu/tagpr/releases/generate-notes",
				func(w http.ResponseWriter, req *http.Request) {
					var body struct {
						TagName         string `json:"tag_name"`
						PreviousTagName string `json:"previous_tag_name"`
					}
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					notesTags = append(notesTags, body.TagName)
					notesBases = append(notesBases, body.PreviousTagName)
					json.NewEncoder(w).Encode(map[string]string{"name": "release", "body": "notes"})
				})
			mux.HandleFunc("/repos/Songmu/tagpr/releases",
				func(w http.ResponseWriter, req *http.Request) {
					if req.Method == http.MethodGet {
						fmt.Fprint(w, "[]")
						return
					}
					createCalls++
					if createCalls == 1 {
						http.Error(w, "temporary failure", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]any{"tag_name": tt.tag, "draft": false})
				})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)
			client, err := github.NewClient(github.WithURLs(new(srv.URL+"/"), nil))
			if err != nil {
				t.Fatal(err)
			}
			cfg := newTestConfig(tt.versionFile)
			cfg.release = new("true")
			cfg.tagPrefix = new(tt.prefix)
			cfg.vPrefix = new(strings.HasPrefix(strings.TrimPrefix(tt.tag, prefix), "v"))
			if tt.calendar {
				cfg.calendarVersioning = new("true")
			}
			tp := &tagpr{
				c: &commander{
					gitPath: "git", dir: r.dir, outStream: io.Discard, errStream: io.Discard,
				},
				gh: client, cfg: cfg, owner: "Songmu", repo: "tagpr",
				runOptions:          runOptions{Mode: executionModeAuto},
				normalizedTagPrefix: prefix,
			}
			if err := tp.Run(context.Background()); err == nil {
				t.Fatal("Run() expected the first release creation to fail")
			}
			if err := tp.Run(context.Background()); err != nil {
				t.Fatalf("Run() retry failed: %v", err)
			}
			wantTags := append([]string{tt.tag}, tt.aliases...)
			if tt.baseTag != "" {
				wantTags = append(wantTags, tt.baseTag)
			}
			slices.Sort(wantTags)
			if got := r.git("tag", "-l"); got != strings.Join(wantTags, "\n") {
				t.Errorf("retry created unexpected tags: %q", got)
			}
			firstNotesTag := tt.tag
			if tt.tagged {
				firstNotesTag += "-notes"
			}
			if !reflect.DeepEqual(notesBases, []string{tt.baseTag, tt.baseTag}) ||
				!reflect.DeepEqual(notesTags, []string{firstNotesTag, tt.tag + "-notes"}) {
				t.Errorf("notes bases=%v, tags=%v; want the original base and retry tag", notesBases, notesTags)
			}
			t.Setenv("GITHUB_OUTPUT", "")
			if err := os.Unsetenv("GITHUB_OUTPUT"); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			tp.out = &output
			tp.runOptions = runOptions{Mode: executionModePlan}
			if err := tp.Run(context.Background()); err != nil {
				t.Fatalf("Run() in plan mode after release failed: %v", err)
			}
			var plan releasePlan
			if err := json.Unmarshal(output.Bytes(), &plan); err != nil {
				t.Fatal(err)
			}
			if plan.Tag != tt.tag || plan.BaseTag != tt.baseTag || plan.CommitSHA != targetSHA {
				t.Errorf("repeated plan changed the released target or baseline: %#v", plan)
			}
			if len(tt.aliases) > 0 {
				if got := tp.latestSemverTag(tt.baseTag, tt.tag); got != "" {
					t.Errorf("excluded full tags left a floating alias as the baseline: %q", got)
				}
				r.git("push", "origin", "main")
				r.git("push", "--force", "origin", plan.CommitSHA+":refs/tags/"+tt.aliases[0])
				tp.runOptions = runOptions{Mode: executionModeApply, Plan: &plan}
				if err := tp.Run(context.Background()); err != nil {
					t.Fatalf("Run() in apply mode with floating aliases failed: %v", err)
				}
				r.git("push", "--force", "origin", plan.CommitSHA+":refs/tags/"+tt.baseTag)
				if err := tp.Run(context.Background()); err == nil ||
					!strings.Contains(err.Error(), "failed to fetch release branch and tags") {
					t.Fatalf("Run() error = %v, want a moved immutable tag to be rejected", err)
				}
				if got := r.git("rev-parse", "refs/tags/"+tt.baseTag+"^{commit}"); got != baseSHA {
					t.Errorf("fetch overwrote immutable tag %s: %s", tt.baseTag, got)
				}
				r.git("tag", "-d", tt.baseTag)
				pr := &github.PullRequest{
					Head: &github.PullRequestBranch{
						Ref: new(branchPrefix + branchSafePrefix(prefix) + baseVersion),
					},
				}
				baseTag, err := tp.releasePullRequestBaseTag(pr)
				if err != nil || baseTag != tt.aliases[1] {
					t.Fatalf("shorthand-only base = %q, err=%v; want %q", baseTag, err, tt.aliases[1])
				}
				if got := tp.latestSemverTag(tt.tag); got != baseTag {
					t.Errorf("shorthand-only latest tag = %q, want %q", got, baseTag)
				}
				tp.runOptions = runOptions{Mode: executionModeAuto}
				if err := tp.Run(context.Background()); err == nil ||
					!strings.Contains(err.Error(), "base tag is not an ancestor of the release boundary") {
					t.Fatalf("Run() error = %v, want the changed release boundary to be rejected", err)
				}
			}
		})
	}
}
func TestApplyReleasePlanRejectsAbbreviatedSHA(t *testing.T) {
	r := newTestRepo(t, "")
	baseSHA := r.git("rev-parse", "HEAD")
	r.merge("merge")
	tp, _ := newTestTagpr(t, r, newTestConfig("-"))
	plan := releasePlan{
		Tag:                     "v0.1.1",
		CommitSHA:               r.git("rev-parse", "HEAD")[:12],
		ReleaseNotesBoundarySHA: baseSHA,
		PullRequestNumber:       1,
		BaseTag:                 "v0.1.0",
	}
	if err := tp.applyReleasePlan(context.Background(), plan); err == nil {
		t.Fatal("applyReleasePlan() expected an error")
	}
}

func TestApplyReleasePlanRejectsNonFinalPRCommit(t *testing.T) {
	r := newTestRepo(t, "")
	t.Chdir(r.dir)
	baseSHA := r.merge("rebase")
	r.git("push", "origin", "main")
	tp, _ := newTestTagpr(t, r, newTestConfig("-"))
	plan := releasePlan{
		Tag:                     "v0.1.1",
		CommitSHA:               r.git("rev-parse", "HEAD^"),
		BaseTag:                 "v0.1.0",
		PullRequestNumber:       1,
		ReleaseNotesBoundarySHA: baseSHA,
	}
	err := tp.applyReleasePlan(context.Background(), plan)
	if err == nil || !strings.Contains(err.Error(), "not the final merge commit") {
		t.Fatalf("applyReleasePlan() error = %v, want a non-final commit error", err)
	}
	if got := r.git("tag", "-l", plan.Tag); got != "" {
		t.Errorf("rejected plan created tag %q", got)
	}
}

func TestApplyCalverPlanValidatesVersionAndRestoresCheckout(t *testing.T) {
	for _, versionFile := range []string{"version.go", "", "-"} {
		t.Run("versionFile="+versionFile, func(t *testing.T) {
			r := newTestRepo(t, "")
			t.Chdir(r.dir)
			r.commit("version.go", `const version = "2026.1101.0"`, "base version")
			r.git("tag", "v2026.1101.0")
			baseSHA := r.git("rev-parse", "HEAD")
			r.commit("version.go", `const version = "2026.1102.0"`, "release version")
			targetSHA := r.git("rev-parse", "HEAD")
			r.git("push", "origin", "main")
			cfg := newTestConfig(versionFile)
			cfg.calendarVersioning = new("true")
			tp, _ := newTestTagpr(t, r, cfg)
			plan := releasePlan{
				Tag:                     "v2026.1102.0",
				CommitSHA:               targetSHA,
				BaseTag:                 "v2026.1101.0",
				PullRequestNumber:       1,
				ReleaseNotesBoundarySHA: baseSHA,
			}
			advancedSHA := r.advanceMain(
				"version.go", `const version = "2026.1103.0"`, "advance after release")
			r.git("push", "origin", "main")
			restoreRef := "main"
			if versionFile == "" {
				r.git("checkout", advancedSHA)
				restoreRef = "HEAD"
			}
			assertCheckout := func() {
				t.Helper()
				if got := r.git("rev-parse", "--abbrev-ref", "HEAD"); got != restoreRef {
					t.Errorf("checkout = %s, want %s", got, restoreRef)
				}
				if got := r.git("rev-parse", "HEAD"); got != advancedSHA {
					t.Errorf("HEAD = %s, want %s", got, advancedSHA)
				}
			}
			if versionFile != "-" {
				mismatched := plan
				mismatched.Tag = "v2026.1104.0"
				err := tp.applyReleasePlan(context.Background(), mismatched)
				if err == nil || !strings.Contains(err.Error(), "planned tag changed") {
					t.Fatalf("mismatched version error = %v, want a changed-tag error", err)
				}
				assertCheckout()
			}
			if versionFile == "version.go" {
				cfg.versionFile = new("missing.go")
				if err := tp.applyReleasePlan(context.Background(), plan); err == nil {
					t.Fatal("missing version file should fail")
				}
				assertCheckout()
				cfg.versionFile = new(versionFile)
			}
			if err := tp.applyReleasePlan(context.Background(), plan); err != nil {
				t.Fatalf("applyReleasePlan() failed: %v", err)
			}
			assertCheckout()
			if got := r.git("rev-parse", plan.Tag); got != targetSHA {
				t.Errorf("tag = %s, want %s", got, targetSHA)
			}
		})
	}
}

func TestValidateCalverTagRequiresConfiguredPrefix(t *testing.T) {
	tests := []struct {
		tag     string
		vPrefix bool
		wantErr bool
	}{
		{tag: "tools/v2026.1102.0", vPrefix: true},
		{tag: "v2026.1102.0", vPrefix: true, wantErr: true},
		{tag: "tools/2026.1102.0", vPrefix: true, wantErr: true},
		{tag: "tools/v2026.1102.0", vPrefix: false, wantErr: true},
		{tag: "tools/2026.1102.0", vPrefix: false},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			cfg := newTestConfig("-")
			cfg.vPrefix = new(tt.vPrefix)
			tp := &tagpr{cfg: cfg, normalizedTagPrefix: "tools/"}
			if err := tp.validateCalverTag(tt.tag); (err != nil) != tt.wantErr {
				t.Fatalf("validateCalverTag() error = %v, wantErr=%t", err, tt.wantErr)
			}
		})
	}
}

func TestCompleteReleaseRetriesAfterReleaseCreationFailure(t *testing.T) {
	for _, baseTag := range []string{"", "v0.1.0"} {
		t.Run("baseTag="+baseTag, func(t *testing.T) {
			r := newTestRepo(t, "")
			baseSHA := r.merge("merge")
			targetSHA := r.git("rev-parse", "HEAD")
			r.git("tag", "v0.1.1", targetSHA)
			r.git("push", "origin", "refs/tags/v0.1.1:refs/tags/v0.1.1")

			var createAttempts int
			var releaseCreated bool
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/Songmu/tagpr/releases/generate-notes",
				func(w http.ResponseWriter, req *http.Request) {
					var notesReq struct {
						TagName         string  `json:"tag_name"`
						PreviousTagName *string `json:"previous_tag_name"`
					}
					if err := json.NewDecoder(req.Body).Decode(&notesReq); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					body := "https://github.com/Songmu/tagpr/commits/" + notesReq.TagName
					if notesReq.PreviousTagName != nil {
						body = "https://github.com/Songmu/tagpr/compare/" + *notesReq.PreviousTagName + "..." + notesReq.TagName
					}
					json.NewEncoder(w).Encode(map[string]string{
						"name": notesReq.TagName,
						"body": "**Full Changelog**: " + body,
					})
				})
			mux.HandleFunc("/repos/Songmu/tagpr/releases/tags/v0.1.1",
				func(w http.ResponseWriter, req *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if !releaseCreated {
						w.WriteHeader(http.StatusNotFound)
						json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
						return
					}
					json.NewEncoder(w).Encode(map[string]any{
						"tag_name": "v0.1.1",
						"draft":    false,
					})
				})
			mux.HandleFunc("/repos/Songmu/tagpr/releases",
				func(w http.ResponseWriter, req *http.Request) {
					if req.Method == http.MethodGet {
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, "[]")
						return
					}
					createAttempts++
					var releaseReq struct {
						Name string `json:"name"`
						Body string `json:"body"`
					}
					if err := json.NewDecoder(req.Body).Decode(&releaseReq); err != nil {
						t.Error(err)
					}
					if releaseReq.Name != "v0.1.1" || strings.Contains(releaseReq.Body, "v0.1.1-notes") {
						t.Errorf("recovered release uses synthetic tag: %#v", releaseReq)
					}
					w.Header().Set("Content-Type", "application/json")
					if createAttempts == 1 {
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(map[string]string{"message": "temporary failure"})
						return
					}
					releaseCreated = true
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]any{
						"tag_name": "v0.1.1",
						"draft":    false,
					})
				})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)
			cli, err := github.NewClient(github.WithURLs(new(srv.URL+"/"), nil))
			if err != nil {
				t.Fatal(err)
			}
			cfg := newTestConfig("-")
			cfg.release = new("true")
			tp := &tagpr{
				c: &commander{
					gitPath: "git", dir: r.dir, outStream: io.Discard, errStream: io.Discard,
				},
				gh:    cli,
				cfg:   cfg,
				owner: "Songmu",
				repo:  "tagpr",
			}
			pr := &github.PullRequest{Number: new(1)}
			plan := releasePlan{
				Tag:                     "v0.1.1",
				CommitSHA:               targetSHA,
				ReleaseNotesBoundarySHA: baseSHA,
				PullRequestNumber:       1,
				BaseTag:                 baseTag,
			}

			if err := tp.completeRelease(context.Background(), plan, pr, true); err == nil {
				t.Fatal("completeRelease() expected the first release creation to fail")
			}
			if err := tp.completeRelease(context.Background(), plan, pr, true); err != nil {
				t.Fatalf("completeRelease() retry failed: %v", err)
			}
			if err := tp.completeRelease(context.Background(), plan, pr, true); err != nil {
				t.Fatalf("completeRelease() existing release check failed: %v", err)
			}
			cfg.release = new("draft")
			if err := tp.completeRelease(context.Background(), plan, pr, true); err == nil ||
				!strings.Contains(err.Error(), "want draft=true") {
				t.Fatalf("completeRelease() error = %v, want a published-release state mismatch", err)
			}
			if createAttempts != 2 {
				t.Errorf("release creation attempts = %d, want 2", createAttempts)
			}
		})
	}
}

func TestCompleteReleaseFindsExistingDraftOnLaterPage(t *testing.T) {
	r := newTestRepo(t, "")
	baseSHA := r.merge("merge")
	targetSHA := r.git("rev-parse", "HEAD")
	r.git("tag", "v0.1.1", targetSHA)
	r.git("push", "origin", "refs/tags/v0.1.1:refs/tags/v0.1.1")

	var createCalls int
	var pages []string
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Songmu/tagpr/releases/generate-notes",
		func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"name": "v0.1.1", "body": "notes"})
		})
	mux.HandleFunc("/repos/Songmu/tagpr/releases/tags/v0.1.1",
		func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "not published", http.StatusNotFound)
		})
	mux.HandleFunc("/repos/Songmu/tagpr/releases",
		func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if req.Method == http.MethodPost {
				createCalls++
				json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.1.1", "draft": true})
				return
			}
			page := req.URL.Query().Get("page")
			pages = append(pages, page)
			if page != "2" {
				w.Header().Set("Link", "<"+srv.URL+
					"/repos/Songmu/tagpr/releases?page=2>; rel=\"next\"")
				json.NewEncoder(w).Encode([]map[string]any{{"tag_name": "v0.1.0"}})
				return
			}
			json.NewEncoder(w).Encode([]map[string]any{{"tag_name": "v0.1.1", "draft": true}})
		})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	cli, err := github.NewClient(github.WithURLs(new(srv.URL+"/"), nil))
	if err != nil {
		t.Fatal(err)
	}
	cfg := newTestConfig("-")
	cfg.release = new("draft")
	tp := &tagpr{
		c: &commander{
			gitPath: "git", dir: r.dir, outStream: io.Discard, errStream: io.Discard,
		},
		gh: cli, cfg: cfg, owner: "Songmu", repo: "tagpr",
	}
	plan := releasePlan{
		Tag:                     "v0.1.1",
		CommitSHA:               targetSHA,
		BaseTag:                 "v0.1.0",
		PullRequestNumber:       1,
		ReleaseNotesBoundarySHA: baseSHA,
	}
	for _, release := range []string{"draft", "true"} {
		t.Run("release="+release, func(t *testing.T) {
			cfg.release = new(release)
			pages = nil
			err := tp.completeRelease(context.Background(), plan, &github.PullRequest{}, true)
			if release == "draft" && err != nil {
				t.Fatalf("completeRelease() failed: %v", err)
			}
			if release == "true" && (err == nil || !strings.Contains(err.Error(), "want draft=false")) {
				t.Fatalf("completeRelease() error = %v, want a draft-state mismatch", err)
			}
			if createCalls != 0 || !reflect.DeepEqual(pages, []string{"", "2"}) {
				t.Errorf("createCalls=%d, pages=%v; want no new release and both pages", createCalls, pages)
			}
		})
	}
}
