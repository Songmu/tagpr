# Getting started

This guide installs tagpr in a repository that releases from `main` and uses Semantic
Versioning. Calendar Versioning, monorepos, and maintenance branches use the same basic
release flow with additional configuration.

If the repository already has published versions, a changelog, or release automation,
start with [Adopting tagpr in an existing project](guides/adopting-tagpr.md).

## Prerequisites

- The repository uses GitHub Actions.
- Releases are made from one designated branch, normally `main`.
- The workflow can write repository contents and pull requests.
- For the default setup, release tags follow SemVer, such as `v1.2.3`.

## Add the workflow

Create `.github/workflows/tagpr.yml`:

```yaml
name: tagpr
on:
  push:
    branches: ["main"]

permissions:
  contents: write
  pull-requests: write
  issues: read

jobs:
  tagpr:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v6
      with:
        persist-credentials: false
    - uses: Songmu/tagpr@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The explicit permissions let tagpr push its release PR branch, create and update the
release pull request, inspect merged pull requests, and create tags and GitHub Releases.
`persist-credentials: false` ensures that tagpr uses the token supplied through its
environment for Git operations instead of credentials retained by checkout.

## Enable pull request creation

In the repository, open **Settings > Actions > General > Workflow permissions** and
enable **Allow GitHub Actions to create and approve pull requests**.

This repository setting is required in addition to the workflow's `permissions` block.

## Run tagpr for the first time

Commit the workflow and push it to `main`. The first run:

1. Finds the latest SemVer tag. If none exists, tagpr starts from `v0.0.0` and includes
   changes from the first commit.
2. Creates `.tagpr` and detects a likely version file when no configuration exists.
3. Creates `.github/release.yml` when neither `.github/release.yml` nor
   `.github/release.yaml` exists.
4. Creates a release pull request containing the proposed version and changelog.

Review the generated `.tagpr` file. Confirm that `tagpr.releaseBranch` is correct and
that `tagpr.versionFile` points to every file whose version must change at release time.

For example:

```ini
[tagpr]
    releaseBranch = main
    versionFile = version.go,action.yml
    vPrefix = true
```

## Tag-only releases

If the project does not keep its version in a file, use:

```ini
[tagpr]
    versionFile = -
```

tagpr will still prepare the changelog, create the release pull request, and tag the
merged commit.

## Prepare the first release

The generated release pull request follows later pushes to `main`. Leave it open until
you are ready to release.

Before merging, you can:

- review the proposed version and changelog;
- commit project-specific release changes directly to the release PR branch
  (`tagpr-from-*`);
- edit the configured version file to select an exact version;
- add `tagpr:minor` or `tagpr:major` to change the proposed SemVer bump.

## Choosing the next version

The default proposal is a patch release. Labels configured by `tagpr.majorLabels` and
`tagpr.minorLabels` on merged pull requests can cause tagpr to add a major or minor
label to the release pull request.

On the release pull request:

- `tagpr:major` or `tagpr/major` selects a major bump;
- `tagpr:minor` or `tagpr/minor` selects a minor bump;
- no version label selects a patch bump;
- an edited version file takes precedence over labels.

Dependabot pull request labels are ignored because they describe the dependency's
version change, not the project's release.

See [Versioning and label rules](guides/versioning.md) for custom label mappings and
the complete precedence rules.

## Release

Merge the release pull request. The merge advances `main`, which runs tagpr again.
tagpr recognizes the merged release pull request, tags the merged commit, and creates a
GitHub Release unless `tagpr.release` disables it.

To publish artifacts or deploy at this point, continue with
[Publishing after a release](guides/publish-after-release.md).

## Next steps

- Review the [release flow and design](concepts/release-flow.md).
- Find additional settings in the [configuration index](reference/configuration.md).
- Use the [README troubleshooting guide](../README.md#troubleshooting) if the workflow
  cannot create or update the release pull request.
