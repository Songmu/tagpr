# Release pull request templates

tagpr uses a Go text template to render the complete release pull request. You can
replace the default template with either a file or inline text.

## Output format

The first non-empty rendered line becomes the pull request title. All remaining
rendered content becomes the pull request body.

For example, this template:

```gotemplate
Release {{.NextVersion}}

Merging this pull request will create {{.NextVersion}}.

{{.Changelog}}
```

produces `Release v1.2.3` as the title and uses the remaining Markdown as the body.

## Available variables

| Variable | Description | Example |
| --- | --- | --- |
| `.NextVersion` | Proposed version, including `v` when enabled but excluding `tagPrefix` | `v1.2.3` |
| `.Branch` | Branch used for the release pull request | `tagpr-from-v1.2.2` |
| `.Changelog` | Generated release notes returned by GitHub | `## What's Changed ...` |
| `.TagPrefix` | Normalized monorepo tag prefix without the trailing slash | `tools` |

Use `.TagPrefix` when the same template serves multiple projects:

```gotemplate
{{if .TagPrefix}}[{{.TagPrefix}}] {{end}}Release {{.NextVersion}}

{{.Changelog}}
```

## Template file

Set `tagpr.template` to the template path:

```ini
[tagpr]
    template = .github/tagpr-template.md
```

The path is relative to the repository root when using the GitHub Action, even when the
`.tagpr` configuration is in a subdirectory.

For a monorepo:

```ini
# tools/.tagpr
[tagpr]
    tagPrefix = tools
    template = tools/.github/tagpr-template.md
```

## Inline template

Set `tagpr.templateText` or `TAGPR_TEMPLATE_TEXT` to provide the template directly. A
multiline environment variable is convenient in GitHub Actions:

```yaml
- uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAGPR_TEMPLATE_TEXT: |
      Release {{.NextVersion}}

      This release is prepared from {{.Branch}}.

      {{.Changelog}}
```

When both `tagpr.template` and `tagpr.templateText` are set, the template file takes
precedence.

## Changelog placement

Include `{{.Changelog}}` where GitHub's generated release notes should appear. Omitting
it removes those notes from the release pull request body, but it does not disable
changelog-file updates or GitHub Release generation.

See [Changelog and GitHub Releases](../guides/changelog-and-releases.md) for how those
notes are generated.

## Invalid templates

If tagpr cannot parse the configured template, or rendering fails, it logs the error and
falls back to its built-in default template.
