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
    - uses: Songmu/tagpr@v0
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
    - uses: Songmu/tagpr@v0
      env:
        GITHUB_TOKEN: ${{ secrets.GH_PAT }}
```

## Description
By using `tagpr`, the release flow can be visible and the maintainer can simply merge pull requests to complete the release.

The release can be made easier and clearer because it can be put into a flow where the release is completed by pressing the merge button on a pull request that is automatically created.

## Configuration
Describe the settings in the .tagpr file directly under the repository in gitconfig format. This is automatically created the first time tagpr is run, but feel free to adjust it. The following configuration items are available

### tagpr.releaseBranch
Generally, it is "main." It is the branch for releases. The pcpr tracks this branch,
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

### tagpr.tmplate (Optional)
Pull request template in go template format

### tagpr.release (Optional)
GitHub Release creation behavior after tagging `[true, draft, false]`  
If this value is not set, the release is to be created.

## Outputs for GitHub Actions

The tagpr produces output to be used in conjunction with subsequent GitHub Actions jobs.

- `pull_request`: Information of the pull request created by tagpr in JSON format
- `tag`: Tag strings are output only if the tagpr has tagged

It is useful to see if tag is available and to run tasks after release. The following is an example of running action-update-semver after release.

```yaml
- uses: actions/checkout@v3
- id: tagpr
  uses: Songmu/tagpr@v0
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
- uses: haya14busa/action-update-semver
  if: "steps.tagpr.outputs.tag != ''"
  with:
    tag: ${{ steps.tagpr.outputs.tag }}
```

## Author

[Songmu](https://github.com/Songmu)
