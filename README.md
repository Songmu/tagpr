tagpr
=======

[![Test Status](https://github.com/Songmu/tagpr/workflows/test/badge.svg?branch=main)][actions]
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
    steps:
    - uses: actions/checkout@v3
    - uses: Songmu/tagpr@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

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
    steps:
    - uses: actions/checkout@v3
      with:
        token: ${{ secrets.GH_PAT }}
    - uses: Songmu/tagpr@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GH_PAT }}
```

## Description
By using `tagpr,` the release flow can be made easier and more apparent because it can be put into a flow where the release is completed by pressing the merge button on a pull request that is automatically created.

If there are differences between the last release and the main branch, tagpr generates a pull request for the next release. The tagpr considers a semver tagged commit as a release. It would be standard practice.

You can leave this pull request until you want to make the next release; each time the main branch is updated, this pull request will automatically follow it.

When this pull request is merged, the merge commit is automatically tagged, and GitHub Releases are created simultaneously.

As mentioned at the beginning of this section, the release process becomes simply a matter of pressing the merge button.

In addition, release items will be made into pull requests, allowing for visualization and review of necessary changes at the time of release. This is also important to prevent accidents.

## Versioning Rules
How tagpr proposes the next version number and how to adjust it.

###  How to determine the next version number of candidate
When creating a pull request by tagpr,  the next version number candidate is determined in the following way.

- Conventional Labels: If the merged pull requests for the next release have labels named "major" or "minor," the version is determined accordingly (of course, major has priority).
- If no conventional labels are found, the patch version is incremented.

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
Command to change files just before release.

### tagpr.template (Optional)
Pull request template in go template format

### tagpr.release (Optional)
GitHub Release creation behavior after tagging `[true, draft, false]`  
If this value is not set, the release is to be created.

### tagpr.majorLabels (Optional)
Label of major update targets. Default is [major]

### tagpr.minorLabels (Optional)
Label of minor update targets. Default is [minor]

## GitHub Enterprise
If you are using GitHub Enterprise, use `GH_ENTERPRISE_TOKEN` instead of `GITHUB_TOKEN`.

```yaml
- uses: Songmu/tagpr@v1
  env:
    GH_ENTERPRISE_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Outputs for GitHub Actions

The tagpr produces output to be used in conjunction with subsequent GitHub Actions jobs.

- `pull_request`: Information of the pull request created by tagpr in JSON format
- `tag`: Tag strings are output only if the tagpr has tagged

It is useful to see if tag is available and to run tasks after release. The following is an example of running action-update-semver after release.

```yaml
- uses: actions/checkout@v3
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
