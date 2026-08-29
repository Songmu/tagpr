# Changelog and GitHub Releases

tagpr delegates release-note generation to GitHub. It uses the
[Generate release notes API][github-generate-release-notes-api] rather than
implementing its own pull request categorization rules.

This provides a more useful starting point than copying raw Git history: pull request
titles describe user-visible changes, labels can group them, and maintainers can review
the result before release. tagpr converts the generated Markdown into a
[Keep a Changelog][keep-a-changelog]-style entry for the changelog file.

## Generation flow

When tagpr prepares a release pull request, it calls GitHub with:

- the proposed next tag;
- the previous release tag, when one exists;
- the configured release branch; and
- the release-note configuration path.

GitHub returns generated release notes for the pull requests and contributors between
the two versions. tagpr converts those notes into the changelog entry and includes the
generated notes in the release pull request body.

After the release pull request is merged, tagpr generates the notes again for the final
tag and uses the returned title and body when creating the GitHub Release. Generating
the notes before creating the release also prevents the release pull request itself
from being included as a released change.

## Customize generated notes

GitHub reads `.github/release.yml` or `.github/release.yaml` by default. The file can
define:

- changelog categories and the labels that place pull requests in them;
- labels and authors to exclude from release notes; and
- the label used for uncategorized changes.

See GitHub's
[automatically generated release notes documentation][github-generated-release-notes]
for the complete schema.

If neither default file exists, tagpr creates this minimal configuration on its first
run:

```yaml
changelog:
  exclude:
    labels:
      - tagpr
```

The exclusion keeps tagpr's own release pull request out of the generated change list.

## Use a different configuration path

Set `tagpr.releaseYAMLPath` when each project in a monorepo needs its own release-note
rules:

```ini
# tools/.tagpr
[tagpr]
    tagPrefix = tools
    changelogFile = tools/CHANGELOG.md
    releaseYAMLPath = tools/.github/release.yml
```

The path is relative to the repository root, not to the `.tagpr` file. If the configured
file does not exist on the release branch, create and commit it before setting
`tagpr.releaseYAMLPath`. Unlike the default `.github/release.yml`, a missing custom
configuration path cannot be bootstrapped by the release pull request.

## Control file and release creation

The generated notes are used in several places, while these settings control which
artifacts tagpr writes:

- `tagpr.changelog = false` stops tagpr from creating or updating the changelog file.
- `tagpr.changelogFile` changes the changelog path from its default,
  `CHANGELOG.md`.
- `tagpr.release = true` creates a published GitHub Release.
- `tagpr.release = draft` creates a draft GitHub Release.
- `tagpr.release = false` creates the tag without creating a GitHub Release.

The release pull request body still uses GitHub's generated notes when changelog-file or
GitHub Release creation is disabled.

If release assets must be built after tagging, see
[Coordinating Immutable Releases](immutable-releases.md) before enabling immutable releases.
It explains when to let tagpr prepare a draft and when to delegate release creation to
another workflow.

[github-generated-release-notes]: https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes
[github-generate-release-notes-api]: https://docs.github.com/en/rest/releases/releases#generate-release-notes-content-for-a-release
[keep-a-changelog]: https://keepachangelog.com/
