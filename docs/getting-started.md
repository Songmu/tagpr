# Getting started

This guide installs tagpr in a repository that uses `main` as its release branch and
follows Semantic Versioning. Calendar Versioning, monorepos, and maintenance branches
use the same basic release flow with additional configuration.

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
  workflow_dispatch:

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

> [!CAUTION]
> With the default `GITHUB_TOKEN`, a tag created by tagpr does not trigger another
> workflow. `pull_request` workflows for a release pull request created or updated by
> tagpr are created in an approval-pending state and require approval from a user with
> write access before they run. See
> [Tagging and release](guides/tag-and-release.md) for details and
> alternative workflow layouts.

## Enable pull request creation

In the repository, open **Settings > Actions > General > Workflow permissions** and
enable **Allow GitHub Actions to create and approve pull requests**.

This repository setting is required in addition to the workflow's `permissions` block
when tagpr uses `GITHUB_TOKEN`. A GitHub App token is governed by the App's permissions
instead.

## Run tagpr for the first time

Commit the workflow and push it to `main`. The first run:

1. Finds the latest SemVer tag. If none exists, tagpr starts from `v0.0.0` and includes
   changes from the first commit.
2. Creates `.tagpr` if it does not exist and detects a likely version file when
   `versionFile` is not configured.
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

Set `versionFile` to files that contain the project's version, such as `package.json`
or `Cargo.toml`. tagpr updates the version in these files and includes the changes in
the release pull request. Multiple files can be specified as a comma-separated list;
include every file that should be updated together at release time.

> [!CAUTION]
> Automatic version updates replace only the first occurrence of the current version
> in each file. Confirm in the first release pull request that tagpr updates the
> intended locations.

tagpr also treats each pull request merged into `releaseBranch` as a changelog entry.

### Tag-only releases

If the project does not keep its version in a file, use:

```ini
[tagpr]
    versionFile = -
```

tagpr will still prepare the changelog, create the release pull request, and tag the
merged commit.

## Prepare the first release

The generated release pull request follows later pushes to `main`. Leave it open until
you are ready to start the release.

Before merging, you can:

- review the proposed version and changelog;
- commit project-specific release changes directly to the release PR branch
  (`tagpr-from-*`);
- edit the titles of included pull requests to adjust how they appear in the changelog;
- label included pull requests to adjust their changelog categories;
- add `tagpr:minor` or `tagpr:major` to change tagpr's proposed next version;
- edit the configured version file to select an exact next version.

To update the release pull request without adding another commit to `main`, manually
run tagpr through the `workflow_dispatch` trigger.

## Choosing the next version

The default proposal is a patch release. When a regular pull request merged after the
previous release has a `major` or `minor` label, tagpr adds `tagpr:major` or
`tagpr:minor` to the release pull request. Configure the source label names with
`tagpr.majorLabels` and `tagpr.minorLabels`.

Dependabot pull request labels are ignored because they describe the dependency's
version change, not the project's release.

On the release pull request:

- `tagpr:major` or `tagpr/major` selects a major bump;
- `tagpr:minor` or `tagpr/minor` selects a minor bump;
- no version label selects a patch bump;
- an edited version file takes precedence over labels.

See [Versioning and label rules](guides/versioning.md) for custom label mappings and
the complete precedence rules.

## Tag and start the release

Merge the release pull request. The merge advances `main`, which runs tagpr again.
tagpr recognizes the merged release pull request and tags the merged commit. That tag
starts the project-specific release flow. tagpr also creates a GitHub Release unless
`tagpr.release` disables it.

To run build, packaging, publishing, or deployment steps from the tag, continue with
[Tagging and release](guides/tag-and-release.md).

## Next steps

- Review the [release flow and design](concepts/release-flow.md).
- Find additional settings in the [configuration index](reference/configuration.md).
- Use the [README troubleshooting guide](../README.md#troubleshooting) if the workflow
  cannot create or update the release pull request.
