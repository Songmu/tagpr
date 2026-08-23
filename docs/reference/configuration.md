# Configuration index

tagpr reads `.tagpr` in Git config format. Every setting also has a corresponding
`TAGPR_*` environment variable, and environment variables take precedence over the
configuration file.

```ini
[tagpr]
    releaseBranch = main
    versionFile = version.go
    changelog = true
    release = draft
```

The complete field-by-field descriptions and defaults currently live in the
[README configuration reference](../../README.md#configuration-reference). This page
groups the settings by purpose so you can find the relevant options quickly.

## Path resolution

Relative paths are resolved from tagpr's working directory, not from the directory that
contains `.tagpr`. The GitHub Action changes to `GITHUB_WORKSPACE` before running
tagpr, so configuration paths are relative to the repository root.

This applies to `versionFile`, `changelogFile`, `releaseYAMLPath`, and `template`.
Configured commands also run from the repository root, and automatic version-file
detection scans from there.

## Release target

| Setting | Purpose |
| --- | --- |
| `tagpr.releaseBranch` | Release branch into which the release pull request is merged |
| `tagpr.fixedMajorVersion` | Target a maintenance branch at one major release line |

## Version selection

| Setting | Purpose |
| --- | --- |
| `tagpr.versionFile` | Version file paths, automatic detection, or tag-only mode |
| `tagpr.vPrefix` | Add `v` to SemVer tags |
| `tagpr.majorLabels` | Labels on merged pull requests that imply a major release |
| `tagpr.minorLabels` | Labels on merged pull requests that imply a minor release |
| `tagpr.calendarVersioning` | Enable CalVer and select its format |

See [Versioning and label rules](../guides/versioning.md) for the complete SemVer,
version-file, and CalVer precedence rules.

## Generated files and commands

| Setting | Purpose |
| --- | --- |
| `tagpr.changelog` | Enable or disable changelog updates |
| `tagpr.changelogFile` | Select the changelog path |
| `tagpr.releaseYAMLPath` | Select the generated release-notes configuration |
| `tagpr.command` | Run a command before version files are updated |
| `tagpr.postVersionCommand` | Run a command after version files are updated |

Commands receive `TAGPR_CURRENT_VERSION` and `TAGPR_NEXT_VERSION`.
See [Release preparation commands](../guides/release-commands.md) for execution order,
examples, and file-handling details.

## Release pull request

| Setting | Purpose |
| --- | --- |
| `tagpr.template` | Use a Go template file for the pull request |
| `tagpr.templateText` | Use an inline Go template |
| `tagpr.commitPrefix` | Customize the prefix of tagpr commits |

See [Release pull request templates](templates.md) for the output format, available
variables, and examples.

## GitHub Release

| Setting | Purpose |
| --- | --- |
| `tagpr.release` | Create a published release, a draft, or no GitHub Release |

See [Immutable GitHub Releases](../guides/immutable-releases.md) when a later workflow
must attach assets before publishing.

## Monorepos

Use `tagpr.tagPrefix` to give each independently released project its own tag namespace.
The configuration itself can live in the project's directory:

```yaml
- uses: Songmu/tagpr@v1
  with:
    config: tools/.tagpr
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Paths in that file remain relative to the repository root. Also scope the version file,
changelog, and release-note configuration:

```ini
# tools/.tagpr
[tagpr]
    tagPrefix = tools
    versionFile = tools/package.json
    changelogFile = tools/CHANGELOG.md
    releaseYAMLPath = tools/.github/release.yml
```

This configuration creates tags such as `tools/v1.2.3`.

## Action configuration

The GitHub Action has two inputs:

- `config` selects a configuration file other than `.tagpr`;
- `version` selects the tagpr executable version installed by the action.

It exposes `tag`, `pull_request`, and `base_tag` outputs. See
[Publishing after a release](../guides/publish-after-release.md) for usage.
