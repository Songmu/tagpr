tagpr
=======

[![Test Status](https://github.com/Songmu/tagpr/actions/workflows/test.yaml/badge.svg?branch=main)][actions]
[![MIT License](https://img.shields.io/github/license/Songmu/tagpr)][license]
[![PkgGoDev](https://pkg.go.dev/badge/github.com/Songmu/tagpr)][PkgGoDev]

[actions]: https://github.com/Songmu/tagpr/actions?workflow=test
[license]: https://github.com/Songmu/tagpr/blob/main/LICENSE
[PkgGoDev]: https://pkg.go.dev/github.com/Songmu/tagpr

The `tagpr` clarify the release flow. It automatically creates and updates a pull request for unreleased items, tag them when they are merged, and create releases.

## Synopsis

The `tagpr` is designed to run on github actions.

```yaml
# .github/workflows/tagpr.yml
name: tagpr
on:
  push:
    branches: ["main"]
jobs:
  tagpr:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
      issues: read
    steps:
    - uses: actions/checkout@v5
      with:
        persist-credentials: false
    - uses: Songmu/tagpr@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

To enable pull requests to be created through GitHub Actions, check the "Allow GitHub Actions to create and approve pull requests" box in the "Workflow permissions" section under "Settings > Actions > General" in the repository where you are installing `tagpr`.

If you do not want to use the token provided by GitHub Actions, do the following This is useful if you want to trigger another action with a tag.

ref. <https://docs.github.com/en/actions/security-guides/automatic-token-authentication#using-the-github_token-in-a-workflow>

For simplicity, we include an example of specifying a personal access token here. However, issuing the temporary token in conjunction with the GitHub App would be safer than a personal access token.

```yaml
name: tagpr
on:
  push:
    branches:
    - main
jobs:
  tagpr:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
      issues: read
    steps:
    - uses: actions/checkout@v5
      with:
        token: ${{ secrets.GH_PAT }}
        persist-credentials: false
    - uses: Songmu/tagpr@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GH_PAT }}
```

## Description
By using `tagpr`, the release flow can be made easier and more apparent because it can be put into a flow where the release is completed by pressing the merge button on a pull request that is automatically created.

If there are differences between the last release and the main branch, tagpr generates a pull request for the next release. The tagpr considers a semver tagged commit as a release. It would be standard practice.

You can leave this pull request until you want to make the next release; each time the main branch is updated, this pull request will automatically follow it.

When this pull request is merged, the merge commit is automatically tagged, and GitHub Releases are created simultaneously.

As mentioned at the beginning of this section, the release process becomes simply a matter of pressing the merge button.

In addition, release items will be made into pull requests, allowing for visualization and review of necessary changes at the time of release. This is also important to prevent accidents.

## Versioning Rules
How tagpr proposes the next version number and how to adjust it.

### Semantic Versioning (Default)

####  How to determine the next version number of candidate
When creating or updating the release PR, tagpr computes the next version in the following steps.

1. Find the latest semver tag (respecting `tagpr.tagPrefix`). If no tag exists yet, tagpr assumes the current version is `v0.0.0` and compares from the first commit.
2. Inspect merged PRs since the last release. If any of those PRs have labels listed in `tagpr.majorLabels` or `tagpr.minorLabels` (defaults: `major`, `minor`), tagpr adds `tagpr:major` or `tagpr:minor` to the release PR automatically.
3. Decide the next version from labels on the release PR: `tagpr:major` or `tagpr/major` => major bump, `tagpr:minor` or `tagpr/minor` => minor bump, otherwise patch bump. If both major and minor labels are present, major wins.
4. When calendar versioning is enabled, labels are ignored and the version is date-based.

#### Label behavior and conventions
tagpr uses labels in two layers: merged PRs since the last release, and the release PR itself.

- tagpr always adds the label `tagpr` to its own release PR so it can recognize it later.
- You can change which labels on merged PRs map to major/minor by configuring `tagpr.majorLabels` and `tagpr.minorLabels` (or their environment variable equivalents).
- You can force a major or minor bump by adding `tagpr:major` or `tagpr:minor` to the release PR.

### Calendar Versioning (Optional)
When `tagpr.calendarVersioning` is set to `true` or a format string, tagpr uses date-based versioning.
Labels are ignored, and versions are determined by the release date.
See [tagpr.calendarVersioning](#tagprcalendarversioning-optional) for details.

### How to adjust the next version by yourself
You can adjust the next version number suggested by tagpr directly on the pull request created by tagpr.

There are two ways to do it.

####  Version file
Edit and commit the version file specified in the .tagpr configuration file to describe the next version

####  Conventional labels
Add labels to the pull request like "tagpr:minor" or "tagpr:major." It is helpful to use a flow that does not use version files.

If there is a discrepancy between the version file and the conventional labels at the time of merging, the specification in the version file takes precedence.

## Configuration
Describe the settings in the .tagpr file directly under the repository in gitconfig format. This is automatically created the first time tagpr is run, but feel free to adjust it. The following configuration items are available

### tagpr.releaseBranch
Generally, it is "main." It is the branch for releases. The tagpr tracks this branch,
creates or updates a pull request as a release candidate, or tags when they are merged.

### tagpr.versionFile
Versioning file containing the semantic version needed to be updated at release.
It will be synchronized with the "git tag".
Often this is a meta-information file such as gemspec, setup.cfg, package.json, etc.
Sometimes the source code file, such as version.go or Bar.pm, is used.
If you do not want to use versioning files but only git tags, specify the "-" string here.
You can specify multiple version files by comma separated strings.

### tagpr.vPrefix
Flag whether or not v-prefix is added to semver when git tagging. (e.g. v1.2.3 if true)  
This is only a tagging convention, not how it is described in the version file.

### tagpr.changelog (Optional)
Flag whether or not changelog is added or changed during the release.

### tagpr.command (Optional)
Command to change files just before release and versioning.

### tagpr.postVersionCommand (Optional)
Command to change files just after versioning.

### tagpr.template (Optional)
Pull request template file in go template format

### tagpr.templateText (Optional)
Pull request template text in go template format

### tagpr.release (Optional)
GitHub Release creation behavior after tagging `[true, draft, false]`  
If this value is not set, the release is to be created.

### tagpr.majorLabels (Optional)
Label(s) of major update targets. Comma-separated. Default is `major`.

### tagpr.minorLabels (Optional)
Label(s) of minor update targets. Comma-separated. Default is `minor`.

### tagpr.commitPrefix (Optional)
Prefix of commit message. Default is "[tagpr]"

### tagpr.tagPrefix (Optional)
Tag prefix for monorepo support (e.g., `tools` produces tags like `tools/v1.2.3`).
This allows managing multiple modules with independent versioning in a single repository.

### tagpr.changelogFile (Optional)
Path to the changelog file. Default is `CHANGELOG.md`.

### tagpr.releaseYAMLPath (Optional)
Path to the GitHub release notes config file used by `gh2changelog`.
If not set, tagpr creates `.github/release.yml` on first run if neither `.github/release.yml` nor `.github/release.yaml` exists.

### tagpr.calendarVersioning (Optional)
Use Calendar Versioning (CalVer) instead of Semantic Versioning.
Set to `true` to use the default format (`YYYY.MM0D.MICRO`), or specify a custom format string directly.
Labels for major/minor are ignored when this option is enabled.

Available format tokens (see https://calver.org):
- Year: `YYYY` (4-digit), `YY` (2-digit), `0Y` (zero-padded 2-digit)
- Month: `MM` (no padding), `0M` (zero-padded)
- Week: `WW` (no padding), `0W` (zero-padded)
- Day: `DD` (no padding), `0D` (zero-padded)
- Micro: `MICRO` (auto-incrementing patch number for same date)

Examples:
- `true` or `"YYYY.MM0D.MICRO"` → `v2026.1203.0` (Dec 3rd, 2026)
- `"YYYY.0M.MICRO"` → `v2026.01.0`
- `"YY.0M0D.MICRO"` → `v26.0123.0`

## GitHub Enterprise
If you are using GitHub Enterprise, use `GH_ENTERPRISE_TOKEN` instead of `GITHUB_TOKEN`.

```yaml
- uses: Songmu/tagpr@v1
  env:
    GH_ENTERPRISE_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Inputs for GitHub Actions

### config (Optional)
A path to the tagpr configuration file.
If not specified, it will be ".tagpr" in the repository root.

## Environment variables
When running `tagpr.command` or `tagpr.postVersionCommand`, tagpr exports the following environment variables:

- `TAGPR_CURRENT_VERSION`: the current version tag (e.g., `v1.2.3`)
- `TAGPR_NEXT_VERSION`: the next version tag (e.g., `v1.3.0`)

## Outputs for GitHub Actions

The tagpr produces output to be used in conjunction with subsequent GitHub Actions jobs.

- `pull_request`: Information of the pull request created by tagpr in JSON format
- `tag`: Tag strings are output only if the tagpr has tagged
- `base_tag`: The base semver tag for comparison, empty if no previous tag exists

It is useful to see if tag is available and to run tasks after release. The following is an example of running action-update-semver after release.

```yaml
- uses: actions/checkout@v5
  with:
    persist-credentials: false
- id: tagpr
  uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
- uses: haya14busa/action-update-semver@v1
  if: "steps.tagpr.outputs.tag != ''"
  with:
    tag: ${{ steps.tagpr.outputs.tag }}
```

## Author

[Songmu](https://github.com/Songmu)
