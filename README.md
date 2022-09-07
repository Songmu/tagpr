tagpr
=======

[![Test Status](https://github.com/Songmu/tagpr/workflows/test/badge.svg?branch=main)][actions]
[![MIT License](https://img.shields.io/github/license/Songmu/tagpr)][license]
[![PkgGoDev](https://pkg.go.dev/badge/github.com/Songmu/tagpr)][PkgGoDev]

[actions]: https://github.com/Songmu/tagpr/actions?workflow=test
[license]: https://github.com/Songmu/tagpr/blob/main/LICENSE
[PkgGoDev]: https://pkg.go.dev/github.com/Songmu/tagpr

The `tagpr` automatically creates and updates a pull request for unreleased items, tag them when they are merged, and create releases.

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
    - uses: Songmu/tagpr@main
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

If you do not want to use the token provided by GitHub Actions, do the following This is useful if you want to trigger another action with a tag.
It would be safer to issue the token in conjunction with the GitHub App instead of a personal access token.

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
    - uses: Songmu/tagpr@main
      env:
        GITHUB_TOKEN: ${{ secrets.GH_PAT }}
```

## Description
By using `tagpr`, the release flow can be visible and the maintainer can simply merge pull requests to complete the release.

## Configuration

Describe the settings in the .tagpr file directly under the repository. This is automatically created the first time tagpr is run, but feel free to adjust it. The following configuration items are available

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

## Author

[Songmu](https://github.com/Songmu)
