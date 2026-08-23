# Publishing after a release

tagpr creates the version tag and exposes outputs that downstream steps can use. Choose
between publishing in the tagpr workflow and triggering a separate workflow.

## Publish in the same workflow

The `tag` output is non-empty only when tagpr creates a tag. Use it as the condition for
a publishing or deployment step:

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
  run: ./scripts/publish "${{ steps.tagpr.outputs.tag }}"
```

This is the simplest option when publishing can run in the same permissions and
environment as tagpr.

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

Use a GitHub App installation token when the tag must trigger a separate workflow. The
token must be used by both checkout and tagpr:

```yaml
- name: Generate token
  id: app-token
  uses: actions/create-github-app-token@v3
  with:
    app-id: ${{ secrets.APP_ID }}
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

The GitHub App must be installed on the repository and granted the requested
permissions.

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
