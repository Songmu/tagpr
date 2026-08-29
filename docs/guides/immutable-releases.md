# Coordinating Immutable Releases

tagpr creates a tag and a published GitHub Release by default. Repositories that enable
[immutable releases][github-immutable-releases] must finish building and attaching
assets before publishing the release: after publication, the associated tag cannot be
moved and release assets cannot be modified or deleted.

GitHub recommends creating a draft release, attaching every asset, and publishing the
draft only when it is complete. Choose which tool owns that sequence before enabling
immutable releases.

## Choose the release owner

tagpr supports two patterns for coordinating with a later build or publishing workflow:

| Configuration | Use when | Release owner |
| --- | --- | --- |
| `tagpr.release = draft` | You want tagpr to generate the release title and notes | tagpr creates the draft; the later workflow adds assets and publishes it |
| `tagpr.release = false` | You want a strict boundary at tag creation | The later workflow creates, populates, and publishes the release |

Do not leave `tagpr.release = true` when a later workflow needs to upload assets to an
immutable release. That setting publishes the release immediately, so it is already
immutable when the later workflow runs.

In either pattern, make the downstream operation accept an explicit tag and remain safe
to rerun. It should update the draft for that exact tag rather than create another tag
or a second release.

## Let tagpr prepare a draft release

Set `release = draft` to keep tagpr's generated title and release notes:

```ini
[tagpr]
    release = draft
```

After tagpr creates a tag, a later step can build the assets, upload them to the existing
draft, and publish it:

```yaml
- id: tagpr
  uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

- name: Build release assets
  if: steps.tagpr.outputs.tag != ''
  run: ./scripts/build-release "${{ steps.tagpr.outputs.tag }}"

- name: Upload assets and publish the draft
  if: steps.tagpr.outputs.tag != ''
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAG: ${{ steps.tagpr.outputs.tag }}
  run: |
    gh release upload "$TAG" dist/project.tar.gz dist/checksums.txt --clobber
    gh release edit "$TAG" --draft=false
```

The upload must succeed before the command publishes the draft. Keep publishing in the
same workflow when using `GITHUB_TOKEN`; tags created with that token do not normally
trigger another workflow. To use a separate tag-triggered workflow, provide tagpr with
a token that can trigger workflows.

The separate workflow can start before the draft exists because tagpr pushes the tag
before it creates the GitHub Release. Retry the draft lookup until it becomes available,
or dispatch the publishing workflow only after the tagpr job completes. Changing the
token makes the workflow trigger possible, but does not by itself prevent this race.
See
[Publishing after a release](publish-after-release.md#github_token-constraints) for the
authentication and workflow-layout tradeoffs.

Use `--clobber` only for assets that the build can reproduce from the same tag. It lets
a retry replace assets uploaded by a partially completed earlier attempt. Monitor for
drafts that remain unpublished after a failed or cancelled workflow, and rerun the
operation for their explicit tags.

### Reuse the draft with GoReleaser

GoReleaser v2.5 or later can populate the draft created by tagpr. Pair the tagpr setting
above with:

```yaml
# .goreleaser.yml
release:
  use_existing_draft: true
```

Without `release = draft`, tagpr publishes too early for an immutable release.
Without `use_existing_draft: true`, GoReleaser may try to create a separate release
instead of adding assets to tagpr's draft. The complete coordination pattern is
described in
[Coordinating tagpr and GoReleaser with immutable releases][tagpr-goreleaser-article].

## Let another workflow create the release

Set `release = false` when tagpr should own only the release pull request and tag:

```ini
[tagpr]
    release = false
```

The downstream workflow then owns release notes, assets, publication, and recovery. For
example, accept a tag input for manual recovery and fall back to the pushed tag during a
normal tag-triggered run:

```yaml
- name: Create a draft and attach assets
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAG: ${{ inputs.tag || github.ref_name }}
  run: |
    if ! gh release view "$TAG" >/dev/null 2>&1; then
      gh release create "$TAG" --draft --generate-notes --verify-tag
    fi
    gh release upload "$TAG" \
      dist/project.tar.gz dist/checksums.txt \
      --clobber

- name: Publish the completed release
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAG: ${{ inputs.tag || github.ref_name }}
  run: gh release edit "$TAG" --draft=false
```

This separation is useful when an existing release tool or workflow already controls
release-note generation. Ensure that the workflow is triggered reliably: a tag created
by tagpr with `GITHUB_TOKEN` does not normally start a tag-triggered workflow. The
[separate workflow guidance](publish-after-release.md#trigger-a-separate-workflow)
explains how to use a GitHub App installation token instead.

The example detects and reuses an existing draft instead of blindly running
`gh release create` again. Keep asset generation deterministic and publish only after
every required upload has completed.

[github-immutable-releases]: https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases
[tagpr-goreleaser-article]: https://songmu.jp/riji/entry/2025-09-05-coordinate-tagpr-and-goreleaser-with-immutable-releases.html
