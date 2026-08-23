# Publishing after a release

tagpr creates the version tag and exposes outputs that downstream steps can use. Choose
between publishing in the tagpr workflow and triggering a separate workflow.

## `GITHUB_TOKEN` constraints

The repository's `GITHUB_TOKEN` is the simplest credential to use with tagpr because
GitHub creates it automatically for each workflow run. However, events created with
`GITHUB_TOKEN` do not normally start another workflow run. This affects tagpr in two
places:

- a tag created by tagpr does not trigger a workflow configured with `on.push.tags`;
- workflows on the release PR created by tagpr do not run automatically.

GitHub allows workflows on pull requests created by `github-actions[bot]` to run after
[a user with write access approves them][bot-pr-approval]. This makes the release PR
workflows available with `GITHUB_TOKEN`, but adds a manual approval step.

There are two ways to run publishing or deployment automatically after tagpr creates a
tag:

| Layout | Advantage | Tradeoff |
| --- | --- | --- |
| Publish in the tagpr workflow | Uses `GITHUB_TOKEN` without additional credentials | Keeps tagpr and publishing in the same workflow |
| Trigger a separate tag workflow | Keeps release responsibilities in separate workflows | Requires a token that can trigger workflows |

## Publish in the same workflow

The `tag` output is non-empty only when tagpr creates a tag. Use it as the condition for
a publishing or deployment step in the same workflow:

```yaml
- uses: actions/checkout@v6
  with:
    persist-credentials: false
- id: tagpr
  uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
- name: Publish
  if: steps.tagpr.outputs.tag != ''
  uses: ./.github/actions/release
  with:
    tag: ${{ steps.tagpr.outputs.tag }}
    token: ${{ secrets.GITHUB_TOKEN }}
```

This layout does not need a GitHub App or personal access token. Keeping the publishing
logic in a script or local composite action limits the coupling even though tagpr and
publishing share a workflow. See
[Songmu/ecschedule's tagpr workflow][ecschedule-tagpr] for a complete example.

Other available outputs are:

- `pull_request`: JSON describing the release pull request;
- `base_tag`: the previous tag used as the comparison base, or an empty value for the
  first release.

## Trigger a separate workflow

GitHub does not start another workflow for events created with the repository's
`GITHUB_TOKEN`. As a result, a tag created by tagpr with `GITHUB_TOKEN` does not trigger
a workflow configured with:

```yaml
on:
  push:
    tags:
    - "v*"
```

Supply tagpr with a token other than `GITHUB_TOKEN` when the tag must trigger a separate
workflow. A personal access token works, but a short-lived GitHub App installation
token created by [`actions/create-github-app-token`][create-app-token] is recommended.

The GitHub App must be installed on the repository with these permissions:

- Contents: Read and write
- Pull requests: Read and write
- Issues: Read-only

Creating the App, installing it, and storing its credentials are covered by the
`actions/create-github-app-token` documentation. Once configured, generate the token
and use it for both checkout and tagpr:

```yaml
- name: Generate token
  id: app-token
  uses: actions/create-github-app-token@v3
  with:
    client-id: ${{ secrets.CLIENT_ID }}
    private-key: ${{ secrets.PRIVATE_KEY }}
    permission-contents: write
    permission-pull-requests: write
    permission-issues: read

- uses: actions/checkout@v6
  with:
    token: ${{ steps.app-token.outputs.token }}
    persist-credentials: false

- uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
```

Tags and release PR updates created with this installation token can trigger their
respective workflows without the `GITHUB_TOKEN` restrictions.

## Keep publishing recoverable

Regardless of which workflow layout you choose, make the publishing operation accept an
explicit tag. This allows a failed release job to be rerun or invoked manually without
creating another release tag.

For example, keep the packaging and upload logic in a script or local composite action:

```yaml
- name: Publish
  run: ./.github/scripts/publish "${{ inputs.tag }}"
```

Both the tagpr workflow and a recovery workflow can then call the same operation.

## Security considerations

- Prefer a short-lived GitHub App installation token over a long-lived personal access
  token.
- Grant only the permissions needed by tagpr and the publishing operation.
- Keep `persist-credentials: false` on checkout so credentials are not retained in the
  local Git configuration.
- Pin third-party actions according to your repository's supply-chain policy.

For the action's complete output reference, see the
[README](../../README.md#outputs).

[bot-pr-approval]: https://github.blog/changelog/2026-06-11-bot-created-pull-requests-can-run-workflows-if-approved/
[create-app-token]: https://github.com/actions/create-github-app-token
[ecschedule-tagpr]: https://github.com/Songmu/ecschedule/blob/main/.github/workflows/tagpr.yaml
