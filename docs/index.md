# tagpr documentation

tagpr is a custom GitHub Action that provides an automated workflow centered on Git
tags as the starting point for releases.

It creates a release pull request that collects unreleased changes, then automatically
tags the merge commit when that pull request is merged. By default, it also creates a
GitHub Release and generates a changelog.

Automating the preparation before tagging keeps release contents visible and
reviewable. Automating the required release-time changes also reduces manual work,
dependence on individual maintainers, and mistakes.

![The release PR branch diverges from the release branch and merges back at the tagged release commit](images/release-flow.png)

## Start here

- [Getting started](getting-started.md) explains the complete path from adding the
  workflow to merging the first release pull request.
- [Adopting tagpr in an existing project](guides/adopting-tagpr.md) explains how to
  introduce tagpr to an existing project.
- [Release flow and design](concepts/release-flow.md) explains the lifecycle of the
  release pull request and the reasoning behind it.
- [Versioning and label rules](guides/versioning.md) explains SemVer proposals, custom
  labels, version-file precedence, and CalVer behavior.
- [Changelog and GitHub Releases](guides/changelog-and-releases.md) explains how tagpr
  uses GitHub's generated release notes and `.github/release.yml`.
- [Release preparation commands](guides/release-commands.md) explains how to automate
  project-specific file changes before release.
- [Publishing after a release](guides/publish-after-release.md) explains how to publish
  artifacts or deploy after tagpr creates a tag.
- [Coordinating Immutable Releases](guides/immutable-releases.md) explains how to
  attach assets before publication by coordinating tagpr with a later release workflow.
- [Configuration index](reference/configuration.md) groups the available settings by
  purpose and links to the complete reference.
- [Release pull request templates](reference/templates.md) documents title and body
  customization.

## Common goals

| Goal | Documentation |
| --- | --- |
| Install tagpr | [Getting started](getting-started.md) |
| Adopt tagpr in an existing project | [Adoption guide](guides/adopting-tagpr.md) |
| Release without a version file | [Tag-only releases](getting-started.md#tag-only-releases) |
| Select a major or minor version | [Versioning and label rules](guides/versioning.md) |
| Use Calendar Versioning | [Configuration index](reference/configuration.md#version-selection) |
| Release projects in a monorepo | [Configuration index](reference/configuration.md#monorepos) |
| Customize changelog categories | [Changelog and GitHub Releases](guides/changelog-and-releases.md#customize-generated-notes) |
| Customize the release pull request | [Release pull request templates](reference/templates.md) |
| Run project-specific release changes | [Release preparation commands](guides/release-commands.md) |
| Publish from the same workflow | [Publishing after a release](guides/publish-after-release.md#publish-in-the-same-workflow) |
| Trigger a separate workflow | [Publishing after a release](guides/publish-after-release.md#trigger-a-separate-workflow) |
| Publish assets with immutable releases | [Coordinating Immutable Releases](guides/immutable-releases.md) |
| Reuse a tagpr draft with GoReleaser | [Coordinating Immutable Releases](guides/immutable-releases.md#reuse-the-draft-with-goreleaser) |
| Diagnose a setup problem | [README troubleshooting](../README.md#troubleshooting) |

## Reference

The complete configuration, GitHub Action inputs, outputs, environment variables, and
GitHub Enterprise instructions currently remain in the
[README](../README.md#configuration-reference). They will stay there until equivalent
standalone reference pages are complete, so no documentation is lost during the
transition to a documentation site.
