# Tagging and release

tagpr treats the version tag as the starting point of the release flow. After tagging,
a project can build and package artifacts, publish packages and GitHub Release assets,
or deploy applications.

The tag identifies the exact source and version being released. tagpr exposes that tag
as an output for downstream steps, or the tag push can trigger a separate release
workflow.

> [!NOTE]
> When a repository enables immutable releases and a downstream operation adds GitHub
> Release assets, publish the release only after every asset is attached. See
> [Coordinating Immutable Releases](immutable-releases.md) for the
> `tagpr.release = draft` and `tagpr.release = false` coordination patterns.

## `GITHUB_TOKEN` constraints {#github_token-constraints}

The repository's `GITHUB_TOKEN` is the simplest credential to use with tagpr because
GitHub creates it automatically for each workflow run. However, events created with
`GITHUB_TOKEN` [do not normally start another workflow run][github-token-trigger]. This
affects tagpr in two places:

- a tag created by tagpr does not trigger a workflow configured with `on.push.tags`;
- `pull_request` workflows for a release pull request created or updated by tagpr are
  created in an approval-pending state, but do not run until
  [a user with write access approves them][bot-pr-approval].

There are two ways to start the release flow automatically after tagpr creates a tag:

| Layout | Advantage | Tradeoff |
| --- | --- | --- |
| Run release steps in the tagpr workflow | Uses `GITHUB_TOKEN` without additional credentials | Release PR workflows require approval, and release steps share tagpr's workflow permissions and environment |
| Trigger a separate release workflow | Separates release responsibilities and lets tag and release PR workflows run automatically | Requires a token that can trigger workflows |

## Run the release in the same workflow {#publish-in-the-same-workflow}

The `tag` output is non-empty only when tagpr creates a tag. Use it as the condition for
release steps in the same workflow:

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

This layout does not need a GitHub App or personal access token. Keeping the release
logic in a script or local composite action limits the coupling even though tagpr and
the project-specific release steps share a workflow. See
[Songmu/ecschedule's tagpr workflow][ecschedule-tagpr] for a complete example.

Other available outputs are:

- `pull_request`: JSON describing the release pull request;
- `base_tag`: the previous tag used as the comparison base, or an empty value for the
  first release.

## Trigger a separate release workflow {#trigger-a-separate-workflow}

To run the release flow in a workflow configured with `on.push.tags`:

```yaml
on:
  push:
    tags:
    - "v*"
```

Supply tagpr with a token other than `GITHUB_TOKEN` so the tag can trigger that
workflow. A personal access token works, but a short-lived GitHub App installation token
created by [`actions/create-github-app-token`][create-app-token] is recommended.

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

Tag creation and release pull request updates performed with this token can trigger
downstream workflows without the `GITHUB_TOKEN` restrictions.

The repository setting **Allow GitHub Actions to create and approve pull requests**
controls `GITHUB_TOKEN`; GitHub App tokens are instead governed by the App's
permissions.

### Keep the release recoverable

Regardless of which workflow layout you choose, make the release operation accept an
explicit tag. This allows a failed build, publication, or deployment to be rerun or
invoked manually without creating another release tag.

For example, keep the packaging and upload logic in a script or local composite action:

```yaml
- name: Publish
  run: ./.github/scripts/publish "${{ inputs.tag }}"
```

Both the tagpr workflow and a recovery workflow can then call the same release
operation.

## Security considerations

- Prefer a short-lived GitHub App installation token over a long-lived personal access
  token.
- Grant only the permissions needed by tagpr and the release operation.
- Keep `persist-credentials: false` on checkout so credentials are not retained in the
  local Git configuration.
- Pin third-party actions according to your repository's supply-chain policy.

For the action's complete output reference, see the
[README](../../README.md#outputs).

[bot-pr-approval]: https://github.blog/changelog/2026-06-11-bot-created-pull-requests-can-run-workflows-if-approved/
[create-app-token]: https://github.com/actions/create-github-app-token
[ecschedule-tagpr]: https://github.com/Songmu/ecschedule/blob/main/.github/workflows/tagpr.yaml
[github-token-trigger]: https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-when-your-workflow-runs/triggering-a-workflow
