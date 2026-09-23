package tagpr

import (
	"errors"
	"strings"
	"testing"
	"text/template"

	"github.com/google/go-github/v83/github"
)

func renderTestTemplate(t *testing.T, text string, arg *tmplArg) (string, error) {
	t.Helper()
	tmpl, err := template.New(t.Name()).Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	return newPRTmpl(tmpl).Render(arg)
}

func TestPRTmplLazilyEvaluatesChangelog(t *testing.T) {
	tests := map[string]struct {
		text      string
		want      string
		wantCalls int
	}{
		"direct method": {
			text:      `{{.Changelog}}`,
			want:      "notes",
			wantCalls: 1,
		},
		"false branch": {
			text:      `{{if false}}{{.Changelog}}{{end}}Release`,
			want:      "Release",
			wantCalls: 0,
		},
		"unused definition": {
			text:      `{{define "unused"}}{{.Changelog}}{{end}}Release`,
			want:      "Release",
			wantCalls: 0,
		},
		"used definition": {
			text:      `{{define "body"}}{{.Changelog}}{{end}}{{template "body" .}}`,
			want:      "notes",
			wantCalls: 1,
		},
		"variable alias": {
			text:      `{{$root := .}}{{$root.Changelog}}`,
			want:      "notes",
			wantCalls: 1,
		},
		"non-empty operation": {
			text:      `{{index .Changelog 0}}`,
			want:      "110",
			wantCalls: 1,
		},
		"multiple references": {
			text:      `{{.Changelog}}{{.Changelog}}`,
			want:      "notesnotes",
			wantCalls: 1,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			calls := 0
			lazy := &lazyDraftChangelog{load: func() (*draftChangelog, error) {
				calls++
				return &draftChangelog{generatedNotes: "notes"}, nil
			}}
			got, err := renderTestTemplate(t, tt.text, &tmplArg{
				loadChangelog: lazy.GeneratedNotes,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
			if calls != tt.wantCalls {
				t.Errorf("changelog calls = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestPRTmplFallbackLazilyLoadsChangelog(t *testing.T) {
	calls := 0
	got, err := renderTestTemplate(t, `Release {{.Missing}}`, &tmplArg{
		NextVersion: "v1.2.3",
		loadChangelog: func() (string, error) {
			calls++
			return "notes", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "notes") {
		t.Errorf("fallback output does not contain generated notes:\n%s", got)
	}
	if calls != 1 {
		t.Errorf("changelog calls = %d, want 1", calls)
	}
}

func TestPRTmplReturnsChangelogErrorWithoutFallback(t *testing.T) {
	wantErr := errors.New("generate release notes")
	got, err := renderTestTemplate(t, `{{.Changelog}}`, &tmplArg{
		loadChangelog: func() (string, error) {
			return "", wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Render() error = %v, want %v", err, wantErr)
	}
	if got != "" {
		t.Errorf("Render() = %q, want empty output", got)
	}
}

func TestLoadPRTmplParseFailureUsesLazyDefault(t *testing.T) {
	calls := 0
	pt := loadPRTmpl(&config{templateText: github.Ptr(`{{`)})
	got, err := pt.Render(&tmplArg{
		loadChangelog: func() (string, error) {
			calls++
			return "notes", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "notes") {
		t.Errorf("fallback output does not contain generated notes:\n%s", got)
	}
	if calls != 1 {
		t.Errorf("changelog calls = %d, want 1", calls)
	}
}

func TestLazyDraftChangelogCachesResult(t *testing.T) {
	calls := 0
	lazy := &lazyDraftChangelog{load: func() (*draftChangelog, error) {
		calls++
		return &draftChangelog{
			changelog:      "entry",
			generatedNotes: "notes",
		}, nil
	}}

	first, err := lazy.GeneratedNotes()
	if err != nil {
		t.Fatal(err)
	}
	second, err := lazy.GeneratedNotes()
	if err != nil {
		t.Fatal(err)
	}
	result, err := lazy.Load()
	if err != nil {
		t.Fatal(err)
	}
	if first != "notes" || second != "notes" || result.changelog != "entry" {
		t.Errorf("cached result = %q, %q, %q", first, second, result.changelog)
	}
	if calls != 1 {
		t.Errorf("load calls = %d, want 1", calls)
	}
}

func TestLazyDraftChangelogCachesError(t *testing.T) {
	wantErr := errors.New("generate release notes")
	calls := 0
	lazy := &lazyDraftChangelog{load: func() (*draftChangelog, error) {
		calls++
		return nil, wantErr
	}}

	for range 2 {
		if _, err := lazy.GeneratedNotes(); !errors.Is(err, wantErr) {
			t.Fatalf("GeneratedNotes() error = %v, want %v", err, wantErr)
		}
	}
	if calls != 1 {
		t.Errorf("load calls = %d, want 1", calls)
	}
}
