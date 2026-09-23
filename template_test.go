package tagpr

import (
	"testing"
	"text/template"

	"github.com/google/go-github/v83/github"
)

func TestPRTmplUsesChangelog(t *testing.T) {
	tests := map[string]struct {
		text string
		want bool
	}{
		"direct field": {
			text: `Release {{.NextVersion}}

{{.Changelog}}`,
			want: true,
		},
		"conditional field": {
			text: `{{if .Changelog}}{{.Changelog}}{{end}}`,
			want: true,
		},
		"assigned field": {
			text: `{{$notes := .Changelog}}{{$notes}}`,
			want: true,
		},
		"associated template": {
			text: `{{define "body"}}{{.Changelog}}{{end}}{{template "body" .}}`,
			want: true,
		},
		"comment": {
			text: `{{/* .Changelog */}}Release {{.NextVersion}}`,
			want: false,
		},
		"literal text": {
			text: `Release .Changelog`,
			want: false,
		},
		"other field": {
			text: `Release {{.NextVersion}}`,
			want: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpl, err := template.New(name).Parse(tt.text)
			if err != nil {
				t.Fatal(err)
			}
			if got := newPRTmpl(tmpl).UsesChangelog(); got != tt.want {
				t.Errorf("UsesChangelog() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestPRTmplPrepareFallsBackToDefault(t *testing.T) {
	tmpl, err := template.New("invalid field").Parse(`Release {{.Missing}}`)
	if err != nil {
		t.Fatal(err)
	}
	pt := newPRTmpl(tmpl)
	pt.Prepare(&tmplArg{NextVersion: "v1.2.3"})
	if !pt.UsesChangelog() {
		t.Error("UsesChangelog() = false after render fallback, want true")
	}
}

func TestLoadPRTmplParseFailureFallsBackToDefault(t *testing.T) {
	pt := loadPRTmpl(&config{templateText: github.Ptr(`{{`)})
	if !pt.UsesChangelog() {
		t.Error("UsesChangelog() = false after parse fallback, want true")
	}
}

func TestNeedsDraftReleaseNotes(t *testing.T) {
	withChangelog, err := template.New("with").Parse(`{{.Changelog}}`)
	if err != nil {
		t.Fatal(err)
	}
	withoutChangelog, err := template.New("without").Parse(`Release {{.NextVersion}}`)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		changelog *bool
		tmpl      *template.Template
		want      bool
	}{
		"default changelog setting": {
			tmpl: withoutChangelog,
			want: true,
		},
		"enabled changelog": {
			changelog: github.Ptr(true),
			tmpl:      withoutChangelog,
			want:      true,
		},
		"disabled changelog with template reference": {
			changelog: github.Ptr(false),
			tmpl:      withChangelog,
			want:      true,
		},
		"disabled changelog without template reference": {
			changelog: github.Ptr(false),
			tmpl:      withoutChangelog,
			want:      false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &config{changelog: tt.changelog}
			if got := needsDraftReleaseNotes(cfg, newPRTmpl(tt.tmpl)); got != tt.want {
				t.Errorf("needsDraftReleaseNotes() = %t, want %t", got, tt.want)
			}
		})
	}
}
