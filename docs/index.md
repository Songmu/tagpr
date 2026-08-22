# tagpr documentation

tagpr prepares a release pull request for unreleased changes, then tags the merged
commit and optionally creates a GitHub Release. It makes release preparation automated
and repeatable while keeping the release decision visible and reviewable.

## Start here

- [Getting started](getting-started.md) explains the complete path from adding the
  workflow to merging the first release pull request.
- [Adopting tagpr in an existing project](guides/adopting-tagpr.md) explains how to
  choose a release baseline and transition existing changelog and publishing automation.
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
| Diagnose a setup problem | [README troubleshooting](../README.md#troubleshooting) |

## Reference

The complete configuration, GitHub Action inputs, outputs, environment variables, and
GitHub Enterprise instructions currently remain in the
[README](../README.md#configuration-reference). They will stay there until equivalent
standalone reference pages are complete, so no documentation is lost during the
transition to a documentation site.
