# Release preparation commands

`tagpr.command` and `tagpr.postVersionCommand` automate project-specific file changes
in the release pull request. Use them for work that should be reviewed and committed
before the release tag is created, such as regenerating metadata, synchronizing version
declarations, or updating dependency files.

These commands prepare the release pull request. They do not run after the release is
tagged and are not intended for publishing packages or deploying applications. Use the
action's `tag` output for those tasks.

## Execution order

When tagpr creates or updates a release pull request, it performs the relevant work in
this order:

1. Determine the current and proposed next versions.
2. Run `tagpr.command`.
3. Update every configured version file.
4. Run `tagpr.postVersionCommand`.
5. Create the release-note configuration when needed.
6. Generate and update the changelog when enabled.
7. Commit the resulting changes to the release pull request branch.

Both hooks run again whenever tagpr refreshes the release pull request after the release
branch advances. Commands should therefore be deterministic and safe to run repeatedly.

## Choose the appropriate hook

Use `tagpr.command` when the script should run before tagpr changes version files:

```ini
[tagpr]
    command = go generate ./...
```

Use `tagpr.postVersionCommand` when the script needs to read the proposed version from
the already-updated version files:

```ini
[tagpr]
    postVersionCommand = ./scripts/update-release-metadata
```

Both hooks know the proposed version through environment variables, so a pre-version
command can still generate next-version content without reading a version file.

## Version environment variables

tagpr adds these variables to the command environment:

| Variable | Description | Example |
| --- | --- | --- |
| `TAGPR_CURRENT_VERSION` | Current version, including `v` when enabled | `v1.2.3` |
| `TAGPR_NEXT_VERSION` | Proposed next version, including `v` when enabled | `v1.3.0` |

In a monorepo, these values do not include `tagpr.tagPrefix`. For example, a release
whose full tag is `tools/v1.3.0` receives `TAGPR_NEXT_VERSION=v1.3.0`.

Example script:

```sh
#!/bin/sh
set -eu

printf '%s\n' "$TAGPR_NEXT_VERSION" > generated/release-version.txt
git add generated/release-version.txt
```

## Working directory and paths

Commands run from tagpr's working directory. The GitHub Action runs tagpr from
`GITHUB_WORKSPACE`, so commands normally run from the repository root even when the
selected `.tagpr` file is in a subdirectory.

For a monorepo configuration at `tools/.tagpr`, include the project path in the command:

```ini
# tools/.tagpr
[tagpr]
    command = ./tools/scripts/prepare-release
    postVersionCommand = go run ./tools/cmd/update-metadata
```

## Files included in the release pull request

Changes to tracked files are detected and included in the release pull request. If a
command creates a new untracked file, the command must stage it with `git add` so tagpr
can include it.

Keep generated output inside the repository and avoid making unrelated working-tree
changes. The command may modify or delete more than one file.

## Command output and failures

Standard output and standard error are written to the tagpr log.

In the current implementation, a non-zero command exit status is not propagated as a
tagpr error. The workflow can therefore continue after a command failure. Commands
should validate their output, and maintainers should inspect the resulting release pull
request and action log before merging.

## Release after tagging

Do not use these hooks to upload packages based on a tag that does not exist yet. To run
the release flow only after tagpr creates the tag, use the action's `tag` output as
described in [Tagging and release](tag-and-release.md).
