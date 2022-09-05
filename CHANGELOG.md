# Changelog

## [v0.1.2](https://github.com/Songmu/tagpr/compare/v0.1.1...v0.1.2) - 2022-09-05
- strict commit detection logic for tagging targets, just in case by @Songmu in https://github.com/Songmu/tagpr/pull/82

## [v0.1.1](https://github.com/Songmu/tagpr/compare/v0.1.0...v0.1.1) - 2022-09-04
- adjust internal interfaces of commander by @Songmu in https://github.com/Songmu/tagpr/pull/79
- fix version file detection in perl by @Songmu in https://github.com/Songmu/tagpr/pull/81

## [v0.1.0](https://github.com/Songmu/tagpr/compare/v0.0.15...v0.1.0) - 2022-09-03
- update dependency by @Songmu in https://github.com/Songmu/tagpr/pull/76
- update README.md by @Songmu in https://github.com/Songmu/tagpr/pull/78

## [v0.0.15](https://github.com/Songmu/tagpr/compare/v0.0.14...v0.0.15) - 2022-08-31
- enhance docs by @Songmu in https://github.com/Songmu/tagpr/pull/72
- adjust template args by @Songmu in https://github.com/Songmu/tagpr/pull/74
- introduce github.com/Songmu/gh2changelog by @Songmu in https://github.com/Songmu/tagpr/pull/75

## [v0.0.14](https://github.com/Songmu/tagpr/compare/v0.0.13...v0.0.14) - 2022-08-28
- fix version file detection in releasing by @Songmu in https://github.com/Songmu/tagpr/pull/70

## [v0.0.13](https://github.com/Songmu/tagpr/compare/v0.0.12...v0.0.13) - 2022-08-28
- add actions.yml to support GitHub Actions by @Songmu in https://github.com/Songmu/tagpr/pull/63
- support to specify multiple version files by comma separated string in conf by @Songmu in https://github.com/Songmu/tagpr/pull/65
- adjust bumping version file behavior by @Songmu in https://github.com/Songmu/tagpr/pull/66
- remove generated comment in CHANGELOG.md by @Songmu in https://github.com/Songmu/tagpr/pull/67
- rename tool name to tagpr from rcpr by @Songmu in https://github.com/Songmu/tagpr/pull/68

## [v0.0.12](https://github.com/Songmu/tagpr/compare/v0.0.11...v0.0.12) - 2022-08-27
- adjust default pull request body by @Songmu in https://github.com/Songmu/tagpr/pull/52
- fix remote name detection by @Songmu in https://github.com/Songmu/tagpr/pull/54
- change rc branch naming by @Songmu in https://github.com/Songmu/tagpr/pull/55
- separate out tagging function by @Songmu in https://github.com/Songmu/tagpr/pull/57
- add convention label variations by @Songmu in https://github.com/Songmu/tagpr/pull/58
- define semv.GuessNext to clarify access scope by @Songmu in https://github.com/Songmu/tagpr/pull/59
- add .github/release.yml automatically when it doesn't exist by @Songmu in https://github.com/Songmu/tagpr/pull/60
- specify a command to change files just before release by config by @Songmu in https://github.com/Songmu/tagpr/pull/61
- configurable pull request template by @Songmu in https://github.com/Songmu/tagpr/pull/62

## [v0.0.11](https://github.com/Songmu/tagpr/compare/v0.0.10...v0.0.11) - 2022-08-21
- fix config key to tagpr.vPrefix from tagpr.v-prefix by @Songmu in https://github.com/Songmu/tagpr/pull/50
- reload version file after cherry-picking process by @Songmu in https://github.com/Songmu/tagpr/pull/51

## [v0.0.10](https://github.com/Songmu/tagpr/compare/v0.0.9...v0.0.10) - 2022-08-20
- config tagpr.v-prefix by @Songmu in https://github.com/Songmu/tagpr/pull/45

## [v0.0.9](https://github.com/Songmu/tagpr/compare/v0.0.8...v0.0.9) - 2022-08-20
- implement configuration file by @Songmu in https://github.com/Songmu/tagpr/pull/41
- skip version file detection with document files by @Songmu in https://github.com/Songmu/tagpr/pull/43

## [v0.0.8](https://github.com/Songmu/tagpr/compare/v0.0.7...v0.0.8) - 2022-08-18
- refine version file detection by @Songmu in https://github.com/Songmu/tagpr/pull/39

## [v0.0.7](https://github.com/Songmu/tagpr/compare/v0.0.6...v0.0.7) - 2022-08-18
- fix process around date when updating changelog by @Songmu in https://github.com/Songmu/tagpr/pull/37

## [v0.0.6](https://github.com/Songmu/tagpr/compare/v0.0.5...v0.0.6) - 2022-08-18
- create as many changelogs as possible if missing by @Songmu in https://github.com/Songmu/tagpr/pull/35

## [v0.0.5](https://github.com/Songmu/tagpr/compare/v0.0.4...v0.0.5) - 2022-08-17
- use fi.Mode().IsRegular() by @Songmu in https://github.com/Songmu/tagpr/pull/32
- implement updating CHANGELOG.md process by @Songmu in https://github.com/Songmu/tagpr/pull/34

## [v0.0.4](https://github.com/Songmu/tagpr/compare/v0.0.3...v0.0.4) - 2022-08-17
- create a normal release instead of a draft when tagging by @Songmu in https://github.com/Songmu/tagpr/pull/30

## [v0.0.3](https://github.com/Songmu/tagpr/compare/v0.0.2...v0.0.3) - 2022-08-17
- use personal access token in tagpr by @Songmu in https://github.com/Songmu/tagpr/pull/28
- introduce softprops/aciton-gh-release by @Songmu in https://github.com/Songmu/tagpr/pull/29

## [v0.0.2](https://github.com/Songmu/tagpr/compare/v0.0.1...v0.0.2) - 2022-08-17
- guess the next version from the label name convention by @Songmu in https://github.com/Songmu/tagpr/pull/20
- unshallow in initialize by @Songmu in https://github.com/Songmu/tagpr/pull/22
- retrieve next version after pushing changes into rc branch by @Songmu in https://github.com/Songmu/tagpr/pull/24
- introduce semv struct for representing semver by @Songmu in https://github.com/Songmu/tagpr/pull/25
- create a draft release at the same time it tags by @Songmu in https://github.com/Songmu/tagpr/pull/26

## [v0.0.1](https://github.com/Songmu/tagpr/commits/v0.0.1) - 2022-08-17
- add github.go for github client by @Songmu in https://github.com/Songmu/tagpr/pull/1
- create rc pull request when the default branch proceeded by @Songmu in https://github.com/Songmu/tagpr/pull/2
- dogfooding by @Songmu in https://github.com/Songmu/tagpr/pull/3
- set label to the pull request by @Songmu in https://github.com/Songmu/tagpr/pull/5
- change rc branch naming convention by @Songmu in https://github.com/Songmu/tagpr/pull/6
- adjust auto commit message by @Songmu in https://github.com/Songmu/tagpr/pull/8
- apply the commits added on the RC branch with cherry-pick by @Songmu in https://github.com/Songmu/tagpr/pull/9
- unshallow if a shallow repository by @Songmu in https://github.com/Songmu/tagpr/pull/10
- fix git log by @Songmu in https://github.com/Songmu/tagpr/pull/11
- parse git URL more precise by @Songmu in https://github.com/Songmu/tagpr/pull/12
- fix parseGitURL by @Songmu in https://github.com/Songmu/tagpr/pull/13
- refactor git.go by @Songmu in https://github.com/Songmu/tagpr/pull/14
- set user.email and user.name only if they aren't set by @Songmu in https://github.com/Songmu/tagpr/pull/15
- fix api base handling by @Songmu in https://github.com/Songmu/tagpr/pull/16
- take care of v-prefix or not in tags by @Songmu in https://github.com/Songmu/tagpr/pull/17
- Detect version file and update by @Songmu in https://github.com/Songmu/tagpr/pull/18
- tagging semver to merged tagpr by @Songmu in https://github.com/Songmu/tagpr/pull/19
