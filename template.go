package tagpr

import (
	"bytes"
	"log"
	"text/template"
	"text/template/parse"
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
	NextVersion, Branch, Changelog, TagPrefix string
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

func (pt *prTmpl) Prepare(arg *tmplArg) {
	var b bytes.Buffer
	if err := pt.tmpl.Execute(&b, arg); err != nil {
		log.Printf("failed to render configured template: %s\n", err)
		pt.tmpl = defaultTmpl
	}
}

func (pt *prTmpl) UsesChangelog() bool {
	for _, tmpl := range pt.tmpl.Templates() {
		if tmpl.Tree != nil && nodeUsesChangelog(tmpl.Tree.Root) {
			return true
		}
	}
	return false
}

func needsDraftReleaseNotes(cfg *config, pt *prTmpl) bool {
	return cfg.Changelog() || pt.UsesChangelog()
}

func nodeUsesChangelog(node parse.Node) bool {
	if node == nil {
		return false
	}
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return false
		}
		for _, child := range n.Nodes {
			if nodeUsesChangelog(child) {
				return true
			}
		}
	case *parse.ActionNode:
		if n == nil {
			return false
		}
		return nodeUsesChangelog(n.Pipe)
	case *parse.IfNode:
		if n == nil {
			return false
		}
		return nodeUsesChangelog(n.Pipe) ||
			nodeUsesChangelog(n.List) ||
			nodeUsesChangelog(n.ElseList)
	case *parse.RangeNode:
		if n == nil {
			return false
		}
		return nodeUsesChangelog(n.Pipe) ||
			nodeUsesChangelog(n.List) ||
			nodeUsesChangelog(n.ElseList)
	case *parse.WithNode:
		if n == nil {
			return false
		}
		return nodeUsesChangelog(n.Pipe) ||
			nodeUsesChangelog(n.List) ||
			nodeUsesChangelog(n.ElseList)
	case *parse.TemplateNode:
		if n == nil {
			return false
		}
		return nodeUsesChangelog(n.Pipe)
	case *parse.PipeNode:
		if n == nil {
			return false
		}
		for _, cmd := range n.Cmds {
			if nodeUsesChangelog(cmd) {
				return true
			}
		}
	case *parse.CommandNode:
		if n == nil {
			return false
		}
		for _, arg := range n.Args {
			if nodeUsesChangelog(arg) {
				return true
			}
		}
	case *parse.FieldNode:
		if n == nil {
			return false
		}
		return len(n.Ident) > 0 && n.Ident[0] == "Changelog"
	case *parse.ChainNode:
		if n == nil {
			return false
		}
		return nodeUsesChangelog(n.Node)
	}
	return false
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
