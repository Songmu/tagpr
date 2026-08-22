# Adopting tagpr in an existing project

This guide introduces tagpr to a project that already has versions, tags, changelogs,
and publishing automation. The goal is to make the first tagpr release pull request
predictable without changing or republishing previous releases.

For a new project without an established release process, follow
[Getting started](../getting-started.md) instead.

## Before enabling the workflow

Record the current release state:

- the branch from which releases are made;
- the latest released version and the commit tagged with that version;
- the tag format, including a `v` prefix or a monorepo prefix;
- every file that stores the project version;
- the existing changelog and GitHub release-note configuration; and
- workflows or scripts that create tags, GitHub Releases, packages, or deployments.

Do not enable tagpr while another workflow can create the same version tag or publish
the same artifact. Decide which existing jobs tagpr replaces and which publishing jobs
will continue after tagpr creates a tag.

## 1. Establish the release baseline

tagpr uses the latest matching version tag as the boundary between released and
unreleased changes. Confirm that the latest release tag:

- points to the commit that was actually released;
- follows the SemVer or CalVer scheme that tagpr will use; and
- has the expected `v` and `tagPrefix` values.

For example, a project adopting the default configuration after `v2.4.0` should expect
the first release pull request to contain changes after `v2.4.0` and propose `v2.4.1`
unless labels select a larger increment.

If no matching tag exists, tagpr starts from `v0.0.0` and considers history from the
repository's first commit. Before enabling the workflow, either accept that full
history or create an accurate baseline tag for the last released version. Never move or
replace an existing published tag to establish the baseline.

Monorepo instances only consider tags matching their configured `tagPrefix`.
Maintenance branches can use `fixedMajorVersion` to ignore tags from other major
release lines. See [Versioning and label rules](versioning.md) for the complete
selection rules.

## 2. Commit the configuration first

For an existing project, create `.tagpr` before enabling the workflow rather than
relying on first-run detection. This makes the intended release boundary and generated
files reviewable independently of the first release pull request.

```ini
[tagpr]
    releaseBranch = main
    versionFile = version.go,package.json
    vPrefix = true
    changelogFile = CHANGELOG.md
```

Check these settings in particular:

- `releaseBranch` matches both the workflow trigger and the branch used for releases.
- `versionFile` lists every file tagpr should update. Use `-` for tag-only releases.
- `vPrefix` matches existing tags.
- `tagPrefix` scopes an independently released project in a monorepo.
- `calendarVersioning` matches existing CalVer tags when the project does not use
  SemVer.

Paths are relative to the repository root when using the GitHub Action, even when the
configuration file is in a subdirectory.

The first configured version file should describe the current released version. tagpr
will update it to the proposed next version in the release pull request. Resolve any
disagreement between the version file and the latest release tag before enabling the
workflow.

## 3. Preserve changelog and release-note behavior

tagpr preserves an existing changelog and inserts the generated entry before its first
level-two heading. Review the first generated diff carefully if the file uses a custom
structure. Set `tagpr.changelog = false` if another process must remain responsible for
the changelog.

GitHub generates the release notes used by the pull request, changelog, and GitHub
Release. Existing `.github/release.yml` or `.github/release.yaml` rules continue to
apply. If `tagpr.releaseYAMLPath` specifies a custom path, create and commit that file
to the release branch before enabling the setting.

See [Changelog and GitHub Releases](changelog-and-releases.md) for generation and
configuration details.

## 4. Transition publishing and deployment

Choose one publishing model before the first tagpr release:

- Continue in the tagpr workflow when `steps.tagpr.outputs.tag` is non-empty.
- Keep a separate tag-triggered workflow and supply tagpr with a GitHub App token.

A tag created with the repository's `GITHUB_TOKEN` does not trigger another workflow.
If an existing publishing workflow listens for tag pushes, it will stop running unless
tagpr uses a token that can trigger workflows.

Keep the publishing operation callable with an explicit tag so a failed publication
can be retried without creating another release. See
[Publishing after a release](publish-after-release.md) for both workflow layouts.

## 5. Enable tagpr

Add the workflow from [Getting started](../getting-started.md#add-the-workflow), enable
GitHub Actions to create pull requests, and push the configuration and workflow to the
release branch.

tagpr supports **Create a merge commit** and **Squash and merge** for the release pull
request. It does not support **Rebase and merge**. Confirm that repository merge
settings allow one of the supported methods before the first release.

## Review the first release pull request

Do not merge the first release pull request until these checks match the existing
release process:

- `base_tag` and the changelog start after the intended previous release;
- the proposed version is the expected next version;
- only intended version files and generated files changed;
- existing changelog content remains intact;
- generated release notes use the expected categories and exclusions; and
- publishing or deployment will run exactly once after tagging.

If the history range or generated changes are wrong, leave the pull request open.
Correct `.tagpr`, the baseline, or the release-note configuration on the release branch.
tagpr will regenerate the release proposal. A change to the baseline tag or
`tagPrefix` can produce a different temporary branch; close any superseded release pull
request after confirming the replacement.

You can edit the version file or add `tagpr:minor` or `tagpr:major` on the release pull
request when the proposed SemVer increment needs adjustment. See
[Versioning and label rules](versioning.md) before changing inferred labels.

## Complete the cutover

Merge the release pull request when its contents and downstream workflow are ready.
After tagpr runs again, confirm that:

1. the new tag points to the merged release commit;
2. the GitHub Release and changelog contain the intended changes;
3. package publishing or deployment completed once; and
4. the next unreleased change creates or updates the next release pull request.

After this release succeeds, remove obsolete tagging and release-creation steps from
the previous process. Keep recovery-oriented publishing commands that can accept an
explicit tag.
