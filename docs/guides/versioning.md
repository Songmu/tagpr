# Versioning and label rules

tagpr uses labels to propose Semantic Versioning changes while keeping the final release
version reviewable in the release pull request.

The label rules are intentionally a proposal mechanism, not an inflexible versioning
policy. Whether a change is major, minor, or patch can require project context and
maintainer agreement. The release pull request makes that decision visible, and the
version file remains an explicit final override.

There are two distinct label layers:

1. configurable labels on pull requests merged since the previous release; and
2. fixed `tagpr:major` or `tagpr:minor` labels on the release pull request.

## Default behavior

Without a version label or an edited version file, tagpr proposes a patch release:

| Current version | Release PR labels | Proposed version |
| --- | --- | --- |
| `v1.2.3` | none | `v1.2.4` |
| `v1.2.3` | `tagpr:minor` | `v1.3.0` |
| `v1.2.3` | `tagpr:major` | `v2.0.0` |

`tagpr/minor` and `tagpr/major` are accepted as alternatives to the colon forms. If
both major and minor labels are present, major wins.

tagpr also adds the `tagpr` label to identify its release pull request. That label does
not affect the version.

## Labels on merged pull requests

tagpr inspects pull requests merged after the previous release:

- A label listed in `tagpr.majorLabels` causes tagpr to add `tagpr:major` to the
  release pull request.
- A label listed in `tagpr.minorLabels` causes tagpr to add `tagpr:minor` to the
  release pull request.
- If no configured label matches, the proposal remains a patch release.

The defaults are:

```ini
[tagpr]
    majorLabels = major
    minorLabels = minor
```

Values are comma-separated, trimmed, and matched against label names exactly:

```ini
[tagpr]
    majorLabels = major,breaking-change
    minorLabels = minor,feature,enhancement
```

For this configuration, any matching pull request since the previous release is enough
to select that level. Multiple matching pull requests do not increment the version more
than once.

## Dependabot pull requests

Labels on pull requests created by `dependabot[bot]` are ignored for version selection.
Dependabot commonly applies `major` or `minor` according to a dependency's version
change, which does not necessarily imply the same change to the project itself.

To make a dependency update affect the project version, set the desired label directly
on the release pull request or edit the version file.

## Labels on the release pull request

The release pull request's `tagpr:major` and `tagpr:minor` labels directly control the
SemVer proposal. You can add one manually when merged pull requests do not carry
version labels.

Labels inferred from merged pull requests are added again when tagpr updates the
release pull request. If an inferred level is not appropriate, first correct the source
pull request labels, then remove the corresponding `tagpr:major` or `tagpr:minor` label
from the release pull request. Correcting the source first prevents tagpr from adding
the inferred label again on its next update. Alternatively, select the exact version in
the version file.

## Version-file precedence

When `tagpr.versionFile` is configured, tagpr writes the proposed version into that file
and reads it again after the release pull request is merged. Therefore, an explicit
version committed to the version file takes precedence over release pull request
labels.

With multiple version files, the first configured file determines the final tag and all
configured files are updated as part of release preparation:

```ini
[tagpr]
    versionFile = version.go,action.yml
```

Paths are relative to the repository root when using the GitHub Action.

## Tag-only releases

Set `tagpr.versionFile = -` when Git tags are the only version source:

```ini
[tagpr]
    versionFile = -
```

In this mode, the labels on the release pull request determine the final SemVer tag.
The default is patch, and major takes precedence over minor.

## Calendar Versioning

When `tagpr.calendarVersioning` is enabled, tagpr calculates the next version from the
configured calendar format. Major and minor labels are ignored.

```ini
[tagpr]
    calendarVersioning = YYYY.0M0D.MICRO
```

With a configured version file, tagpr calculates the next CalVer value when it prepares
or refreshes the release pull request and writes the value to that file. Merging the
pull request tags the stored value, even if it is merged on a later date.

With `tagpr.versionFile = -`, there is no stored proposal. tagpr calculates the CalVer
value after merge when it creates the tag. Existing matching CalVer tags determine the
next `MICRO` value in both modes.

See [the `tagpr.calendarVersioning` reference](../../README.md#tagprcalendarversioning-optional)
for the available format tokens.

## Monorepos

With `tagpr.tagPrefix`, version selection is scoped to the independently released
project:

- only tags with the configured prefix are considered;
- merged-change history is limited to the directory represented by the prefix; and
- the release pull request is matched to the same prefix.

```ini
# tools/.tagpr
[tagpr]
    tagPrefix = tools
    versionFile = tools/package.json
    majorLabels = major,breaking-change
    minorLabels = minor,feature
```

This instance proposes versions from `tools/v*` tags and changes under `tools/`.

## Precedence summary

For SemVer releases:

1. An explicit version in the configured version file determines the final tag.
2. Otherwise, `tagpr:major` or `tagpr/major` on the release pull request selects major.
3. Otherwise, `tagpr:minor` or `tagpr/minor` selects minor.
4. Otherwise, tagpr selects patch.

Configured labels on merged pull requests feed into steps 2 and 3 by adding the
corresponding label to the release pull request. With CalVer enabled, the calendar
format replaces this label-based decision.
