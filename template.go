package tagpr

import (
	"bytes"
	"errors"
	"log"
	"text/template"
)

const defaultTmplStr = `{{if .TagPrefix}}[{{.TagPrefix}}] {{end}}Release for {{.NextVersion}}

This pull request is for the next release as {{.NextVersion}} created by [tagpr](https://github.com/Songmu/tagpr). Merging it will tag {{.NextVersion}} to the merge commit and create a GitHub release.

You can modify this branch "{{.Branch}}" directly before merging if you want to change the next version number or other files for the release.

<details>
<summary>How to change the next version as you like</summary>

There are two ways to do it.

- Version file
    - Edit and commit the version file specified in the .tagpr configuration file to describe the next version
    - If you want to use another version file, edit the configuration file.
- Labels convention
    - Add labels to this pull request like "tagpr:minor" or "tagpr:major"
    - If no conventional labels are added, the patch version is incremented as is.
</details>

---
{{.Changelog}}`

var defaultTmpl *template.Template

func init() {
	var err error
	defaultTmpl, err = template.New("pull request template").Parse(defaultTmplStr)
	if err != nil {
		log.Fatal(err)
	}
}

type tmplArg struct {
	NextVersion, Branch, TagPrefix string
	loadChangelog                  func() (string, error)
}

type changelogTemplateError struct {
	err error
}

func (e *changelogTemplateError) Error() string {
	return e.err.Error()
}

func (e *changelogTemplateError) Unwrap() error {
	return e.err
}

func (arg *tmplArg) Changelog() (string, error) {
	if arg.loadChangelog == nil {
		return "", nil
	}
	changelog, err := arg.loadChangelog()
	if err != nil {
		return "", &changelogTemplateError{err: err}
	}
	return changelog, nil
}

func newPRTmpl(tmpl *template.Template) *prTmpl {
	if tmpl == nil {
		tmpl = defaultTmpl
	}
	return &prTmpl{tmpl: tmpl}
}

func loadPRTmpl(cfg *config) *prTmpl {
	if t := cfg.Template(); t != "" {
		tmpl, err := template.ParseFiles(t)
		if err == nil {
			return newPRTmpl(tmpl)
		}
		log.Printf("parse configured template failed: %s\n", err)
	} else if t := cfg.TemplateText(); t != "" {
		tmpl, err := template.New("templateText").Parse(t)
		if err == nil {
			return newPRTmpl(tmpl)
		}
		log.Printf("parse configured template failed: %s\n", err)
	}
	return newPRTmpl(nil)
}

type prTmpl struct {
	tmpl *template.Template
}

func unwrapChangelogTemplateError(err error) error {
	var changelogErr *changelogTemplateError
	if errors.As(err, &changelogErr) {
		return changelogErr.err
	}
	return nil
}

func (pt *prTmpl) Render(arg *tmplArg) (string, error) {
	var b bytes.Buffer
	err := pt.tmpl.Execute(&b, arg)
	if err != nil {
		if changelogErr := unwrapChangelogTemplateError(err); changelogErr != nil {
			return "", changelogErr
		}
		log.Printf("failed to render configured template: %s\n", err)
		b.Reset()
		// fallback to default template
		err = defaultTmpl.Execute(&b, arg)
		if changelogErr := unwrapChangelogTemplateError(err); changelogErr != nil {
			return "", changelogErr
		}
	}
	return b.String(), err
}
