package tagpr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/v93/github"
)

type releasePRTree struct {
	BaseTree string              `json:"base_tree"`
	Entries  []*github.TreeEntry `json:"tree"`
}

type releasePRRequests struct {
	trees       []releasePRTree
	commits     []github.Commit
	createdRefs []github.CreateRef
	updatedRefs []github.UpdateRef
	refPaths    []string
	pull        map[string]json.RawMessage
	pullMethod  string
	stderrAtPR  string
	labels      []string
	notes       []github.GenerateNotesRequest
	featurePRs  []int
	merges      []string
	tempRefs    []string
	stderr      bytes.Buffer
	outputFile  string

	existing       bool
	existingLabels []string
	searchIssues   []*github.Issue
	pullStatus     int
	mergeStatus    map[string]int
	mergeTrees     map[string]string
}

// newReleasePRTagpr exercises local Git normally and stubs only the GitHub
// operations used to prepare a release PR. Trees and commits are real objects,
// so publishing the temporary ref also exercises fetch/checkout and cfg.Reload.
func newReleasePRTagpr(t *testing.T, r *testRepo) (*tagpr, *releasePRRequests) {
	t.Helper()
	t.Chdir(r.dir)
	t.Setenv(envGitHubEventName, "")
	t.Setenv(envGitHubEventPath, "")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, name := range []string{
		envConfigFile, envReleaseBranch, envVersionFile, envVPrefix, envChangelog,
		envCommand, envPostVersionCommand, envTemplate, envTemplateText, envRelease,
		envMajorLabels, envMinorLabels, envCommitPrefix, envTagPrefix, envChangelogFile,
		envCalendarVersioning, envReleaseYAMLPath, envFixedMajorVersion,
	} {
		t.Setenv(name, "")
	}
	remote := r.git("remote", "get-url", "origin")
	// Keep the GitHub identity available to gh2changelog without network Git I/O.
	r.git("config", "url."+remote+".insteadOf", "https://github.com/Songmu/tagpr.git")
	r.git("remote", "set-url", "origin", "https://github.com/Songmu/tagpr.git")
	if err := os.MkdirAll(".github", 0755); err != nil {
		t.Fatal(err)
	}
	r.commit(".github/release.yml", "changelog:\n  exclude:\n    labels: [tagpr]\n", "configure notes")
	r.commit(".tagpr", "[tagpr]\nreleaseBranch = main\nversionFile = version.go\nvPrefix = true\nchangelog = false\ntemplateText = Release {{.NextVersion}}\\n\\nBranch {{.Branch}}\n", "configure tagpr")
	r.git("push", "origin", "main")
	cfg, err := newConfig("git")
	if err != nil {
		t.Fatal(err)
	}
	requests := &releasePRRequests{
		mergeStatus: map[string]int{},
		mergeTrees:  map[string]string{},
		outputFile:  filepath.Join(r.dir, "output"),
		searchIssues: []*github.Issue{{
			Number: new(8), User: &github.User{Login: new("developer")},
			Labels: []*github.Label{{Name: "minor"}},
		}},
	}
	t.Setenv("GITHUB_OUTPUT", requests.outputFile)

	// A separate index avoids disturbing the staged changes Run reads.
	gitInput := func(input string, args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = r.dir
		cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(r.dir, ".git", "api-index"))
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	repositoryCommit := func(sha string) *github.RepositoryCommit {
		return &github.RepositoryCommit{
			SHA: new(sha),
			Commit: &github.Commit{
				Message: new(gitInput("", "show", "-s", "--format=%s", sha)),
				Tree:    &github.Tree{SHA: new(gitInput("", "rev-parse", sha+"^{tree}"))},
			},
			Parents: []*github.Commit{{SHA: new(gitInput("", "rev-parse", sha+"^"))}},
		}
	}
	prResponse := func() *github.PullRequest {
		pr := &github.PullRequest{
			Number: new(42), Title: new("Release v0.2.0"), Body: new("updated body"),
			Head: &github.PullRequestBranch{Ref: new("tagpr-from-v0.1.0")},
			Base: &github.PullRequestBranch{Ref: new("main")},
		}
		if requests.existing {
			pr.Labels = append(pr.Labels, &github.Label{Name: "tagpr"})
			for _, label := range requests.existingLabels {
				pr.Labels = append(pr.Labels, &github.Label{Name: label})
			}
		}
		for _, label := range requests.labels {
			pr.Labels = append(pr.Labels, &github.Label{Name: label})
		}
		if requests.pull != nil {
			if err := json.Unmarshal(requests.pull["title"], &pr.Title); err != nil {
				t.Errorf("decode response title: %v", err)
			}
			if err := json.Unmarshal(requests.pull["body"], &pr.Body); err != nil {
				t.Errorf("decode response body: %v", err)
			}
		}
		return pr
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(req.URL.Path, "/repos/Songmu/tagpr")
		encode := func(v any) { _ = json.NewEncoder(w).Encode(v) }
		decode := func(v any) {
			if err := json.NewDecoder(req.Body).Decode(v); err != nil {
				t.Errorf("decode %s %s: %v", req.Method, path, err)
			}
		}
		switch {
		case strings.HasSuffix(path, "/pulls") && strings.HasPrefix(path, "/commits/"):
			// A feature PR association avoids the release-index retry delay.
			encode([]*github.PullRequest{{Number: new(7)}})
		case req.URL.Path == "/search/issues":
			encode(map[string]any{"total_count": len(requests.searchIssues), "items": requests.searchIssues})
		case path == "/issues/7" && req.Method == http.MethodGet:
			requests.featurePRs = append(requests.featurePRs, 7)
			encode(&github.Issue{
				Number: new(7), User: &github.User{Login: new("developer")},
				Labels: []*github.Label{{Name: "minor"}},
			})
		case path == "/pulls" && req.Method == http.MethodGet:
			if req.URL.Query().Get("head") != "Songmu:tagpr-from-v0.1.0" || req.URL.Query().Get("base") != "main" {
				t.Errorf("unexpected release PR query: %s", req.URL.RawQuery)
			}
			if requests.existing {
				pr := prResponse()
				pr.Title, pr.Body = new("old title"), new("User instructions\n\n---\nold notes")
				encode([]*github.PullRequest{pr})
			} else {
				encode([]*github.PullRequest{})
			}
		case (path == "/pulls" && req.Method == http.MethodPost) ||
			(path == "/pulls/42" && req.Method == http.MethodPatch):
			requests.pullMethod = req.Method
			requests.stderrAtPR = requests.stderr.String()
			decode(&requests.pull)
			if requests.pullStatus != 0 {
				w.WriteHeader(requests.pullStatus)
				encode(map[string]string{"message": "PR processing failed"})
				return
			}
			encode(prResponse())
		case path == "/pulls/42" && req.Method == http.MethodGet:
			encode(prResponse())
		case path == "/issues/42/labels" && req.Method == http.MethodPost:
			decode(&requests.labels)
			encode(prResponse().Labels)
		case path == "/releases/generate-notes" && req.Method == http.MethodPost:
			var body github.GenerateNotesRequest
			decode(&body)
			requests.notes = append(requests.notes, body)
			encode(map[string]string{
				"name": "notes",
				"body": "## What's Changed\n* A feature\n\n**Full Changelog**: https://" + req.Host + "/Songmu/tagpr/compare/v0.1.0...v0.2.0",
			})
		case path == "/git/trees" && req.Method == http.MethodPost:
			var body releasePRTree
			decode(&body)
			requests.trees = append(requests.trees, body)
			gitInput("", "read-tree", body.BaseTree)
			for _, entry := range body.Entries {
				if entry.SHA == nil && entry.Content == nil {
					gitInput("", "update-index", "--force-remove", "--", entry.GetPath())
					continue
				}
				sha := gitInput(entry.GetContent(), "hash-object", "-w", "--stdin")
				gitInput("", "update-index", "--add", "--cacheinfo", entry.GetMode(), sha, entry.GetPath())
			}
			encode(&github.Tree{SHA: new(gitInput("", "write-tree"))})
		case path == "/git/commits" && req.Method == http.MethodPost:
			var wire struct {
				Message string   `json:"message"`
				Tree    string   `json:"tree"`
				Parents []string `json:"parents"`
			}
			decode(&wire)
			body := github.Commit{Message: new(wire.Message), Tree: &github.Tree{SHA: new(wire.Tree)}}
			for _, parent := range wire.Parents {
				body.Parents = append(body.Parents, &github.Commit{SHA: new(parent)})
			}
			requests.commits = append(requests.commits, body)
			args := []string{"commit-tree", body.Tree.GetSHA(), "-m", body.GetMessage()}
			for _, parent := range body.Parents {
				args = append(args, "-p", parent.GetSHA())
			}
			body.SHA = new(gitInput("", args...))
			encode(&body)
		case strings.HasPrefix(path, "/commits/"):
			encode(repositoryCommit(strings.TrimPrefix(path, "/commits/")))
		case path == "/merges" && req.Method == http.MethodPost:
			var body github.RepositoryMergeRequest
			decode(&body)
			requests.merges = append(requests.merges, body.Head)
			if body.Base != "tagpr-temp" {
				t.Errorf("merge base = %q, want tagpr-temp", body.Base)
			}
			if status := requests.mergeStatus[body.Head]; status != 0 {
				w.WriteHeader(status)
				encode(map[string]string{"message": "merge failed"})
				return
			}
			tree, ok := requests.mergeTrees[body.Head]
			if !ok {
				t.Errorf("missing merge-result fixture for %s", body.Head)
				http.Error(w, "missing merge-result fixture", http.StatusInternalServerError)
				return
			}
			merged := repositoryCommit(body.Head)
			merged.Commit.Tree.SHA = new(tree)
			encode(merged)
		case path == "/git/refs" && req.Method == http.MethodPost:
			var body github.CreateRef
			decode(&body)
			requests.createdRefs = append(requests.createdRefs, body)
			if body.Ref == "refs/heads/tagpr-temp" {
				requests.tempRefs = append(requests.tempRefs, "create")
			}
			gitInput("", "push", "--force", "origin", body.SHA+":"+body.Ref)
			encode(&github.Reference{Ref: new(body.Ref), Object: &github.GitObject{SHA: new(body.SHA)}})
		case strings.HasPrefix(path, "/git/refs/") && req.Method == http.MethodPatch:
			var body github.UpdateRef
			decode(&body)
			ref := strings.TrimPrefix(path, "/git/refs/")
			requests.updatedRefs = append(requests.updatedRefs, body)
			requests.refPaths = append(requests.refPaths, ref)
			gitInput("", "push", "--force", "origin", body.SHA+":refs/"+ref)
			encode(&github.Reference{Ref: new("refs/" + ref), Object: &github.GitObject{SHA: new(body.SHA)}})
		case path == "/git/refs/heads/tagpr-temp" && req.Method == http.MethodDelete:
			requests.tempRefs = append(requests.tempRefs, "delete")
			if gitInput("", "ls-remote", "origin", "refs/heads/tagpr-temp") == "" {
				w.WriteHeader(http.StatusUnprocessableEntity)
				encode(map[string]string{"message": "ref missing"})
			} else {
				gitInput("", "push", "origin", ":refs/heads/tagpr-temp")
				w.WriteHeader(http.StatusNoContent)
			}
		case strings.HasPrefix(path, "/git/ref/") && req.Method == http.MethodGet:
			ref := strings.TrimPrefix(path, "/git/ref/")
			sha := gitInput("", "rev-parse", "origin/main")
			if ref != "heads/main" {
				if !requests.existing {
					w.WriteHeader(http.StatusNotFound)
					encode(map[string]string{"message": "ref missing"})
					return
				}
				sha = gitInput("", "rev-parse", "tagpr-from-v0.1.0")
			}
			encode(&github.Reference{Ref: new("refs/" + ref), Object: &github.GitObject{SHA: new(sha)}})
		default:
			t.Errorf("unexpected GitHub request: %s %s", req.Method, req.URL)
			http.NotFound(w, req)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client, err := github.NewClient(github.WithURLs(new(srv.URL+"/"), nil))
	if err != nil {
		t.Fatal(err)
	}
	return &tagpr{
		c:       &commander{gitPath: "git", dir: r.dir, outStream: io.Discard, errStream: &requests.stderr},
		gitPath: "git", remoteName: "origin", gh: client, cfg: cfg,
		owner: "Songmu", repo: "tagpr", out: io.Discard,
		runOptions: runOptions{Mode: executionModeAuto},
	}, requests
}

func readReleasePRFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func pullRequestField(t *testing.T, requests *releasePRRequests, field string) string {
	t.Helper()
	var value string
	if err := json.Unmarshal(requests.pull[field], &value); err != nil {
		t.Fatalf("decode PR %s: %v", field, err)
	}
	return value
}

func assertReleasePROutput(t *testing.T, requests *releasePRRequests) {
	t.Helper()
	output := readReleasePRFile(t, requests.outputFile)
	for line := range strings.SplitSeq(output, "\n") {
		if encoded, found := strings.CutPrefix(line, "pull_request="); found {
			var pr github.PullRequest
			if err := json.Unmarshal([]byte(encoded), &pr); err != nil {
				t.Fatal(err)
			}
			if pr.GetNumber() != 42 || pr.GetTitle() != pullRequestField(t, requests, "title") ||
				pr.GetBody() != pullRequestField(t, requests, "body") {
				t.Errorf("output does not contain the refreshed PR: %s", encoded)
			}
			var got, want []string
			for _, label := range pr.Labels {
				got = append(got, label.GetName())
			}
			if requests.existing {
				want = append(want, "tagpr")
				want = append(want, requests.existingLabels...)
			}
			want = append(want, requests.labels...)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("output PR labels = %v, want post-addition labels %v", got, want)
			}
			return
		}
	}
	t.Fatalf("pull_request output missing: %s", output)
}

func TestRunCreatesReleasePR(t *testing.T) {
	r := newTestRepo(t, "package test\nconst Version = \"0.1.0\"\n")
	tp, requests := newReleasePRTagpr(t, r)
	requests.searchIssues = nil
	r.git("config", "-f", ".tagpr", "--unset", configVersionFile)
	var err error
	tp.cfg, err = newConfig("git")
	if err != nil {
		t.Fatal(err)
	}
	r.git("checkout", "-b", "feature")
	r.commit("FEATURE.md", "new feature\n", "add feature")
	r.git("checkout", "main")
	r.git("merge", "--no-ff", "-m", "Merge pull request #7", "feature")
	r.git("push", "origin", "main", "feature:refs/pull/7/head")
	baseSHA := r.git("rev-parse", "HEAD")
	if err := tp.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := readReleasePRFile(t, "version.go"); !strings.Contains(got, `"0.2.0"`) {
		t.Errorf("version file = %q, want minor bump", got)
	}
	if tp.cfg.VersionFile() != "version.go" {
		t.Errorf("detected version file = %q", tp.cfg.VersionFile())
	}
	if requests.pullMethod != http.MethodPost {
		t.Errorf("PR method = %s, want POST", requests.pullMethod)
	}
	for field, want := range map[string]string{
		"title": "Release v0.2.0", "body": "Branch tagpr-from-v0.1.0",
		"base": "main", "head": "Songmu:tagpr-from-v0.1.0",
	} {
		if got := pullRequestField(t, requests, field); got != want {
			t.Errorf("PR %s = %q, want %q", field, got, want)
		}
	}
	if !reflect.DeepEqual(requests.labels, []string{"tagpr:minor", "tagpr"}) {
		t.Errorf("added labels = %v", requests.labels)
	}
	if !reflect.DeepEqual(requests.featurePRs, []int{7}) {
		t.Errorf("merged feature PR lookups = %v", requests.featurePRs)
	}
	if len(requests.commits) != 1 || requests.commits[0].GetMessage() != "[tagpr] prepare for the next release" ||
		len(requests.commits[0].Parents) != 1 ||
		requests.commits[0].Parents[0].GetSHA() != baseSHA {
		t.Fatalf("unexpected release commits: %+v", requests.commits)
	}
	if len(requests.trees) != 1 || requests.trees[0].BaseTree != baseSHA {
		t.Errorf("release trees = %+v, want base %s", requests.trees, baseSHA)
	}
	if len(requests.createdRefs) != 1 || requests.createdRefs[0].Ref != "refs/heads/tagpr-from-v0.1.0" ||
		requests.createdRefs[0].SHA != baseSHA {
		t.Errorf("created refs = %+v", requests.createdRefs)
	}
	if len(requests.updatedRefs) != 1 || requests.updatedRefs[0].Force == nil || !*requests.updatedRefs[0].Force ||
		!reflect.DeepEqual(requests.refPaths, []string{"heads/tagpr-from-v0.1.0"}) {
		t.Fatalf("updated refs = %+v", requests.updatedRefs)
	}
	published := r.git("ls-remote", "origin", "refs/heads/tagpr-from-v0.1.0")
	if !strings.HasPrefix(published, requests.updatedRefs[0].SHA+"\t") {
		t.Errorf("published ref = %q", published)
	}
	if got := r.git("show", requests.updatedRefs[0].SHA+":version.go"); !strings.Contains(got, `"0.2.0"`) {
		t.Errorf("committed version = %q", got)
	}
	output := readReleasePRFile(t, requests.outputFile)
	if !strings.Contains(output, "base_tag=v0.1.0\n") {
		t.Errorf("GitHub output = %q", output)
	}
	assertReleasePROutput(t, requests)
	if len(requests.notes) != 0 {
		t.Errorf("unreferenced disabled changelog generated notes %d times", len(requests.notes))
	}
}

func TestRunUpdatesReleasePROmitsBase(t *testing.T) {
	r := newTestRepo(t, "package test\nconst Version = \"0.1.0\"\n")
	tp, requests := newReleasePRTagpr(t, r)
	requests.existing = true
	requests.existingLabels = []string{"tagpr:major"}
	r.git("push", "origin", "tagpr-from-v0.1.0")
	if err := tp.cfg.set(configTemplateText, "Release {{.NextVersion}}\n\n---\nNew notes"); err != nil {
		t.Fatal(err)
	}
	if err := tp.cfg.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := tp.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests.pullMethod != http.MethodPatch {
		t.Errorf("PR method = %s, want PATCH", requests.pullMethod)
	}
	if _, present := requests.pull["base"]; present {
		t.Errorf("PATCH must omit base: %s", requests.pull["base"])
	}
	if got := pullRequestField(t, requests, "title"); got != "Release v1.0.0" {
		t.Errorf("title = %q", got)
	}
	if got := r.git("show", requests.updatedRefs[0].SHA+":version.go"); !strings.Contains(got, `"1.0.0"`) {
		t.Errorf("existing major label did not override new minor label: %q", got)
	}
	if got := pullRequestField(t, requests, "body"); got != "---\nNew notes" {
		t.Errorf("replacement body = %q", got)
	}
	if !reflect.DeepEqual(requests.labels, []string{"tagpr:minor"}) {
		t.Errorf("added labels = %v, want only new minor label", requests.labels)
	}
	if len(requests.createdRefs) != 0 || len(requests.updatedRefs) != 1 {
		t.Errorf("existing branch refs: creates=%d updates=%d", len(requests.createdRefs), len(requests.updatedRefs))
	}
	assertReleasePROutput(t, requests)
}

func TestRunHookFailuresContinueAndJoinLaterError(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
	}{
		{"PR succeeds", 0},
		{"PR fails", http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRepo(t, "package test\nconst Version = \"0.1.0\"\n")
			tp, requests := newReleasePRTagpr(t, r)
			r.commit("obsolete.md", "obsolete artifact\n", "add obsolete artifact")
			r.commit("empty.md", "old content\n", "add file to clear")
			r.git("push", "origin", "main")
			for key, command := range map[string]string{
				configCommand:            `cat version.go > pre-version; rm obsolete.md; : > empty.md; printf 'pre failed\n' >&2; exit 2`,
				configPostVersionCommand: `cat version.go > post-version; printf 'post failed\n' >&2; exit 3`,
			} {
				if err := tp.cfg.set(key, command); err != nil {
					t.Fatal(err)
				}
			}
			if err := tp.cfg.Reload(); err != nil {
				t.Fatal(err)
			}
			requests.pullStatus = tt.status
			err := tp.Run(context.Background())
			if err == nil {
				t.Fatal("Run succeeded despite hook failures")
			}
			for _, want := range []string{configCommand, "exit status 2", "pre failed", configPostVersionCommand, "exit status 3", "post failed"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("joined error %q does not contain %q", err, want)
				}
			}
			var apiErr *github.ErrorResponse
			if tt.status != 0 {
				if !errors.As(err, &apiErr) || apiErr.Response.StatusCode != tt.status ||
					!strings.Contains(err.Error(), "PR processing failed") {
					t.Errorf("joined error lost API error: %v", err)
				}
			} else {
				if errors.As(err, &apiErr) {
					t.Errorf("successful PR processing returned an API error: %v", err)
				}
				assertReleasePROutput(t, requests)
			}
			if !strings.Contains(readReleasePRFile(t, "pre-version"), `"0.1.0"`) ||
				!strings.Contains(readReleasePRFile(t, "post-version"), `"0.2.0"`) {
				t.Error("hooks did not run on the expected sides of the bump")
			}
			if requests.pullMethod != http.MethodPost || len(requests.updatedRefs) != 1 {
				t.Fatal("hook failures prevented branch/PR processing")
			}
			if tt.status == 0 {
				sha := requests.updatedRefs[0].SHA
				if got := r.git("ls-tree", "--name-only", sha, "--", "obsolete.md"); got != "" {
					t.Errorf("published hook-deleted path still exists: %q", got)
				}
				if got := r.git("ls-tree", "--name-only", sha, "--", "empty.md"); got != "empty.md" {
					t.Errorf("published empty file is missing: %q", got)
				}
				if got := r.git("show", sha+":empty.md"); got != "" {
					t.Errorf("published cleared file content = %q, want empty", got)
				}
			}
			for _, name := range []string{configCommand, configPostVersionCommand} {
				if !strings.Contains(requests.stderrAtPR, "::error title="+name+" failed::") {
					t.Errorf("hook failure was not annotated before PR processing: %s", requests.stderrAtPR)
				}
			}
		})
	}
}

func TestRunGeneratesChangelogLazily(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("changelog=%t", enabled), func(t *testing.T) {
			r := newTestRepo(t, "package test\nconst Version = \"0.1.0\"\n")
			tp, requests := newReleasePRTagpr(t, r)
			r.commit("CHANGELOG.md", "# Changelog\n\n## Older release\n", "existing changelog")
			r.git("push", "origin", "main")
			for key, value := range map[string]string{
				configChangelog:    fmt.Sprint(enabled),
				configTemplateText: "Release {{.NextVersion}}\n\n{{.Changelog}}\n{{.Changelog}}",
			} {
				if err := tp.cfg.set(key, value); err != nil {
					t.Fatal(err)
				}
			}
			if err := tp.cfg.Reload(); err != nil {
				t.Fatal(err)
			}
			if err := tp.Run(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(requests.notes) != 1 {
				t.Fatalf("generated notes %d times, want once", len(requests.notes))
			}
			if requests.notes[0].TagName != "v0.2.0" || requests.notes[0].PreviousTagName == nil ||
				*requests.notes[0].PreviousTagName != "v0.1.0" || requests.notes[0].TargetCommitish == nil ||
				*requests.notes[0].TargetCommitish != "main" {
				t.Errorf("notes request = %+v", requests.notes[0])
			}
			body := pullRequestField(t, requests, "body")
			if strings.Count(body, "* A feature") != 2 || !strings.Contains(body, "compare/v0.1.0...tagpr-from-v0.1.0") {
				t.Errorf("PR changelog = %q", body)
			}
			changelog := readReleasePRFile(t, "CHANGELOG.md")
			if enabled {
				if len(requests.commits) != 2 || requests.commits[1].GetMessage() != "[tagpr] update CHANGELOG.md" {
					t.Fatalf("changelog commits = %+v", requests.commits)
				}
				if !strings.Contains(changelog, "- A feature") || !strings.Contains(changelog, "## Older release") {
					t.Errorf("updated changelog = %q", changelog)
				}
				if got := r.git("show", requests.updatedRefs[0].SHA+":CHANGELOG.md"); got != strings.TrimSpace(changelog) {
					t.Errorf("published changelog = %q, local = %q", got, changelog)
				}
			} else if len(requests.commits) != 1 || changelog != "# Changelog\n\n## Older release\n" {
				t.Error("disabled changelog changed the file or created a changelog commit")
			}
		})
	}
}

func TestRunCherryPicksOldestFirstAndReloads(t *testing.T) {
	r := newTestRepo(t, "package test\nconst Version = \"0.1.0\"\n")
	tp, requests := newReleasePRTagpr(t, r)
	requests.existing = true
	// Include the configured files in the remote PR's manual-commit trees.
	r.git("checkout", "tagpr-from-v0.1.0")
	r.git("checkout", "main", "--", ".tagpr", ".github/release.yml")
	r.git("commit", "-m", "[tagpr] prepare for the next release")
	r.commit("MANUAL.md", "first\n", "first manual change")
	oldest := r.git("rev-parse", "--short", "HEAD")
	r.commit("MANUAL.md", "conflicting\n", "conflicting manual change")
	conflict := r.git("rev-parse", "--short", "HEAD")
	r.commit("custom-version", "1.4.0\n", "override version file")
	r.git("config", "-f", ".tagpr", configVersionFile, "custom-version")
	r.git("config", "-f", ".tagpr", configTagPrefix, "module")
	r.git("config", "-f", ".tagpr", configTemplateText, "[{{.TagPrefix}}] Release {{.NextVersion}}\n\n{{.Branch}}\n{{.Changelog}}")
	r.git("add", ".tagpr")
	r.git("commit", "-m", "customize release")
	newest := r.git("rev-parse", "--short", "HEAD")
	override := r.git("rev-parse", "--short", "HEAD^")
	r.git("push", "origin", "tagpr-from-v0.1.0")
	// Explicit merge-result snapshots retain the generated version bump and
	// accepted manual changes, without importing the skipped conflicting tree.
	r.git("checkout", "-b", "merge-results", "main")
	r.commit("version.go", "package test\nconst Version = \"0.2.0\"\n", "fixture version bump")
	r.commit("MANUAL.md", "first\n", "fixture first merge")
	requests.mergeTrees[oldest] = r.git("rev-parse", "HEAD^{tree}")
	r.commit("custom-version", "1.4.0\n", "fixture version override")
	requests.mergeTrees[override] = r.git("rev-parse", "HEAD^{tree}")
	r.git("checkout", newest, "--", ".tagpr")
	r.git("commit", "-m", "fixture config override")
	requests.mergeTrees[newest] = r.git("rev-parse", "HEAD^{tree}")
	r.git("checkout", "main")
	requests.mergeStatus[conflict] = http.StatusConflict
	if err := tp.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(requests.merges, []string{oldest, conflict, override, newest}) {
		t.Errorf("cherry-pick order = %v", requests.merges)
	}
	type replayTree struct {
		head, tree string
	}
	var replayTrees []replayTree
	for _, commit := range requests.commits {
		if head, ok := strings.CutPrefix(commit.GetMessage(), "cherry-pick: "); ok {
			replayTrees = append(replayTrees, replayTree{head, commit.Tree.GetSHA()})
		}
	}
	wantReplayTrees := []replayTree{
		{oldest, requests.commits[0].Tree.GetSHA()},
		{conflict, requests.mergeTrees[oldest]},
		{override, requests.mergeTrees[oldest]},
		{newest, requests.mergeTrees[override]},
	}
	if !reflect.DeepEqual(replayTrees, wantReplayTrees) {
		t.Errorf("temporary replay trees = %v, want latest accepted trees %v", replayTrees, wantReplayTrees)
	}
	if !reflect.DeepEqual(requests.tempRefs, []string{"delete", "create", "delete"}) {
		t.Errorf("temporary ref lifecycle = %v", requests.tempRefs)
	}
	if r.git("ls-remote", "origin", "refs/heads/tagpr-temp") != "" {
		t.Error("temporary remote ref was not deleted")
	}
	if tp.cfg.VersionFile() != "custom-version" || tp.normalizedTagPrefix != "module/" {
		t.Errorf("configuration was not reloaded: file=%q prefix=%q", tp.cfg.VersionFile(), tp.normalizedTagPrefix)
	}
	if got := pullRequestField(t, requests, "title"); got != "[module] Release v1.4.0" {
		t.Errorf("title after cherry-pick = %q", got)
	}
	if got := pullRequestField(t, requests, "body"); !strings.Contains(got, "tagpr-from-v0.1.0") {
		t.Errorf("reloaded template body = %q", got)
	}
	if len(requests.notes) != 1 || requests.notes[0].TagName != "module/v1.4.0" {
		t.Errorf("notes did not use reloaded prefix/version: %+v", requests.notes)
	}
	if got := r.git("show", requests.updatedRefs[len(requests.updatedRefs)-1].SHA+":custom-version"); got != "1.4.0" {
		t.Errorf("published custom version = %q", got)
	}
	if got := r.git("show", requests.updatedRefs[len(requests.updatedRefs)-1].SHA+":MANUAL.md"); got != "first" {
		t.Errorf("published manual changes = %q, want accepted first change only", got)
	}
	if got := readReleasePRFile(t, "MANUAL.md"); got != "first\n" {
		t.Errorf("checked-out manual changes = %q, want first\\n", got)
	}
}

func TestRunCherryPickNonConflictError(t *testing.T) {
	r := newTestRepo(t, "package test\nconst Version = \"0.1.0\"\n")
	tp, requests := newReleasePRTagpr(t, r)
	requests.existing = true
	r.git("checkout", "tagpr-from-v0.1.0")
	r.commit("MANUAL.md", "manual\n", "manual change")
	sha := r.git("rev-parse", "--short", "HEAD")
	r.git("push", "origin", "tagpr-from-v0.1.0")
	r.git("checkout", "main")
	requests.mergeStatus[sha] = http.StatusInternalServerError
	err := tp.Run(context.Background())
	var apiErr *github.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Run error = %v, want merge API failure", err)
	}
	if !reflect.DeepEqual(requests.merges, []string{sha}) || requests.pullMethod != "" {
		t.Errorf("processing after merge error: merges=%v PR=%s", requests.merges, requests.pullMethod)
	}
}
