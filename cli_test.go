package tagpr

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestRunRejectsInvalidExecutionOptions(t *testing.T) {
	tests := map[string]struct {
		argv    []string
		wantErr string
	}{
		"invalid mode": {
			argv:    []string{"--mode", "invalid"},
			wantErr: "invalid mode",
		},
		"missing apply plan": {
			argv:    []string{"--mode", "apply"},
			wantErr: "--plan is required",
		},
		"plan in plan mode": {
			argv:    []string{"--mode", "plan", "--plan", "{}"},
			wantErr: "--plan is only valid in apply mode",
		},
		"plan in auto mode": {
			argv:    []string{"--plan", "{}"},
			wantErr: "--plan is only valid in apply mode",
		},
		"malformed plan": {
			argv:    []string{"--mode", "apply", "--plan", "{"},
			wantErr: "failed to parse --plan",
		},
		"null plan": {
			argv:    []string{"--mode", "apply", "--plan", "null"},
			wantErr: "--plan must be a JSON object",
		},
		"incomplete plan": {
			argv:    []string{"--mode", "apply", "--plan", `{"tag":"v1.2.3"}`},
			wantErr: "plan.commit_sha is required",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run(context.Background(), tt.argv, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Run() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), []string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}
	if !strings.HasPrefix(stdout.String(), "tagpr v") {
		t.Errorf("version output = %q", stdout.String())
	}
}

func TestRunPlanOutput(t *testing.T) {
	for _, destination := range []string{"stdout", "actions", "api error"} {
		t.Run(destination, func(t *testing.T) {
			r := newTestRepo(t, "")
			t.Chdir(r.dir)
			baseSHA := r.merge("merge")
			targetSHA := r.git("rev-parse", "HEAD")

			mux := http.NewServeMux()
			mux.HandleFunc("/api/v3/repos/Songmu/tagpr/commits/"+targetSHA+"/pulls",
				func(w http.ResponseWriter, req *http.Request) {
					if destination == "api error" {
						http.Error(w, "invalid request", http.StatusBadRequest)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode([]map[string]any{{
						"number":           1,
						"merged_at":        "2026-11-02T00:00:00Z",
						"merge_commit_sha": targetSHA,
						"head":             map[string]string{"ref": "tagpr-from-v0.1.0"},
						"base":             map[string]string{"ref": "main", "sha": baseSHA},
						"labels":           []map[string]string{{"name": "tagpr"}},
					}})
				})
			srv := httptest.NewTLSServer(mux)
			t.Cleanup(srv.Close)
			r.git("remote", "set-url", "origin", srv.URL+"/Songmu/tagpr")
			r.git("config", "github.token", "test-token")
			t.Setenv("GITHUB_TOKEN", "test-token")
			t.Setenv(envReleaseBranch, "main")
			t.Setenv(envVersionFile, "-")
			t.Setenv(envVPrefix, "true")
			t.Setenv(envGitHubEventName, "")
			t.Setenv("GITHUB_OUTPUT", "")
			outputPath := filepath.Join(t.TempDir(), "output")
			if destination == "actions" {
				t.Setenv("GITHUB_OUTPUT", outputPath)
			} else if err := os.Unsetenv("GITHUB_OUTPUT"); err != nil {
				t.Fatal(err)
			}

			var stdout, stderr bytes.Buffer
			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, srv.Client())
			err := Run(ctx, []string{"--mode", "plan"}, &stdout, &stderr)
			if destination == "api error" {
				if err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), "::error") {
					t.Fatalf("Run() error=%v, stdout=%q, stderr=%q; want only stderr diagnostics",
						err, stdout.String(), stderr.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() failed: %v; stderr=%s", err, stderr.String())
			}
			planJSON := stdout.String()
			diagnostics := stderr.String()
			if destination == "actions" {
				b, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatal(err)
				}
				planJSON = ""
				for line := range strings.SplitSeq(string(b), "\n") {
					if name, value, _ := strings.Cut(line, "="); name == "plan" {
						planJSON = value
					}
				}
				diagnostics = stdout.String()
			}
			var got releasePlan
			if err := json.Unmarshal([]byte(planJSON), &got); err != nil {
				t.Fatalf("plan output is not JSON: %v; output=%q", err, planJSON)
			}
			want := releasePlan{
				Tag:                     "v0.1.1",
				CommitSHA:               targetSHA,
				BaseTag:                 "v0.1.0",
				PullRequestNumber:       1,
				ReleaseNotesBoundarySHA: baseSHA,
			}
			if got != want {
				t.Errorf("plan=%#v, want %#v", got, want)
			}
			if !strings.Contains(diagnostics, "::add-mask::") {
				t.Errorf("token masking diagnostic was not routed correctly: %q", diagnostics)
			}
		})
	}
}
