package tagpr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

// newTestTagpr builds a tagpr which talks to a stub GitHub API server. The
// target_commitish passed to the release notes generation API is stored into
// the returned pointer.
func newTestTagpr(t *testing.T, r *testRepo, cfg *config) (*tagpr, *string) {
	t.Helper()
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
