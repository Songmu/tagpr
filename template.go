package tagpr

import (
	"bytes"
	"log"
	"text/template"
)

const defaultTmplStr = `Release for {{.NextVersion}}

This pull request is for the next release as {{.NextVersion}} created by [tagpr](https://github.com/Songmu/tagpr). Merging it will tag {{.NextVersion}} to the merge commit and create a GitHub release.

You can modify this branch "{{.RCBranch}}" directly before merging if you want to change the next version number or other files for the release.

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
	NextVersion, RCBranch, Changelog string
}

func newPRTmpl(tmpl *template.Template) *prTmpl {
	if tmpl == nil {
		tmpl = defaultTmpl
	}
	return &prTmpl{tmpl: tmpl}
}

type prTmpl struct {
	tmpl *template.Template
}

func (pt *prTmpl) Render(arg *tmplArg) (string, error) {
	var b bytes.Buffer
	err := pt.tmpl.Execute(&b, arg)
	if err != nil {
		log.Printf("failed to render configured template: %s\n", err)
		b.Reset()
		// fallback to default template
		err = defaultTmpl.Execute(&b, arg)
	}
	return b.String(), err
}
