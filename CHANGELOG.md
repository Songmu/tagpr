# Changelog

## [v1.1.4](https://github.com/Songmu/tagpr/compare/v1.1.3...v1.1.4) - 2023-10-31
- Fix latest semver tag retrieval for first time setup by @stefafafan in https://github.com/Songmu/tagpr/pull/160

## [v1.1.3](https://github.com/Songmu/tagpr/compare/v1.1.2...v1.1.3) - 2023-10-18
- Fix syntax errors on docs. by @kyo-ago in https://github.com/Songmu/tagpr/pull/150
- Use the official actions/create-github-app-token Action instead of tibdex/github-app-token by @stefafafan in https://github.com/Songmu/tagpr/pull/158
- Consider vPrefix when retrieving the latest semver tag. by @k1LoW in https://github.com/Songmu/tagpr/pull/159

## [v1.1.2](https://github.com/Songmu/tagpr/compare/v1.1.1...v1.1.2) - 2023-01-20
- fix: Bug fixes related to #144 by @toritori0318 in https://github.com/Songmu/tagpr/pull/148

## [v1.1.1](https://github.com/Songmu/tagpr/compare/v1.1.0...v1.1.1) - 2023-01-18
- fix: skip version file detection by @toritori0318 in https://github.com/Songmu/tagpr/pull/145

## [v1.1.0](https://github.com/Songmu/tagpr/compare/v1.0.8...v1.1.0) - 2023-01-15
- Fixing typo in config's `tagpr.tmplate` by @k2tzumi in https://github.com/Songmu/tagpr/pull/140
- feat: Alternative labels for minor and major labels can be specified by @k2tzumi in https://github.com/Songmu/tagpr/pull/142
- update deps by @Songmu in https://github.com/Songmu/tagpr/pull/143

## [v1.0.8](https://github.com/Songmu/tagpr/compare/v1.0.7...v1.0.8) - 2022-12-17
- Fix a typo by @T-Toshiya in https://github.com/Songmu/tagpr/pull/137

## [v1.0.7](https://github.com/Songmu/tagpr/compare/v1.0.6...v1.0.7) - 2022-10-15
- SetOutput in GitHub Actions with new way by @Songmu in https://github.com/Songmu/tagpr/pull/134

## [v1.0.6](https://github.com/Songmu/tagpr/compare/v1.0.5...v1.0.6) - 2022-10-09
- [bugfix] Care the case if you do not use a version file by @Songmu in https://github.com/Songmu/tagpr/pull/131

## [v1.0.5](https://github.com/Songmu/tagpr/compare/v1.0.4...v1.0.5) - 2022-10-09
- don't update versionFile configuration if .tagpr file already exists by @Songmu in https://github.com/Songmu/tagpr/pull/127

## [v1.0.4](https://github.com/Songmu/tagpr/compare/v1.0.3...v1.0.4) - 2022-10-07
- SearchIssues related testing and refactoring by @k2tzumi in https://github.com/Songmu/tagpr/pull/121
- Add manifest.json as a priority item in the version file search. by @Songmu in https://github.com/Songmu/tagpr/pull/126

## [v1.0.3](https://github.com/Songmu/tagpr/compare/v1.0.2...v1.0.3) - 2022-10-02
- [fix] reset query variable after requesting while fetching squashed issues by @Songmu in https://github.com/Songmu/tagpr/pull/122

## [v1.0.2](https://github.com/Songmu/tagpr/compare/v1.0.1...v1.0.2) - 2022-09-29
- Correction of version number detection over 2 digits by @Songmu in https://github.com/Songmu/tagpr/pull/119

## [v1.0.1](https://github.com/Songmu/tagpr/compare/v1.0.0...v1.0.1) - 2022-09-25
- declare outputs properly in action.yml by @Songmu in https://github.com/Songmu/tagpr/pull/116

## [v1.0.0](https://github.com/Songmu/tagpr/compare/v0.3.5...v1.0.0) - 2022-09-23
- ready for v1 by @Songmu in https://github.com/Songmu/tagpr/pull/112
- add Versioning Rules section into README.md by @Songmu in https://github.com/Songmu/tagpr/pull/114

## [v0.3.5](https://github.com/Songmu/tagpr/compare/v0.3.4...v0.3.5) - 2022-09-22
- introduce tibdex/github-app-token by @Songmu in https://github.com/Songmu/tagpr/pull/110

## [v0.3.4](https://github.com/Songmu/tagpr/compare/v0.3.3...v0.3.4) - 2022-09-18
- fix retrievingVersionFromFile by @Songmu in https://github.com/Songmu/tagpr/pull/108

## [v0.3.3](https://github.com/Songmu/tagpr/compare/v0.3.2...v0.3.3) - 2022-09-17
- use install.sh in action.yml by @Songmu in https://github.com/Songmu/tagpr/pull/106

## [v0.3.2](https://github.com/Songmu/tagpr/compare/v0.3.1...v0.3.2) - 2022-09-17
- add vN tag after uploading artifacts by @Songmu in https://github.com/Songmu/tagpr/pull/103
- update dependencies by @Songmu in https://github.com/Songmu/tagpr/pull/105

## [v0.3.1](https://github.com/Songmu/tagpr/compare/v0.3.0...v0.3.1) - 2022-09-17
- update docs by @Songmu in https://github.com/Songmu/tagpr/pull/99
- adjust label convention by @Songmu in https://github.com/Songmu/tagpr/pull/101
- fix label convention by @Songmu in https://github.com/Songmu/tagpr/pull/102

## [v0.3.0](https://github.com/Songmu/tagpr/compare/v0.2.0...v0.3.0) - 2022-09-17
- add convention labels from labels of existing pull requests by @Songmu in https://github.com/Songmu/tagpr/pull/93
- refine around type config by @Songmu in https://github.com/Songmu/tagpr/pull/94
- add tagpr.release setting to adjust creating GitHub Release behavior by @Songmu in https://github.com/Songmu/tagpr/pull/95
- true/false/draft for tapgr.release config by @Songmu in https://github.com/Songmu/tagpr/pull/96
- GitHub Actions friendly outputs by @Songmu in https://github.com/Songmu/tagpr/pull/97

## [v0.2.0](https://github.com/Songmu/tagpr/compare/v0.1.3...v0.2.0) - 2022-09-06
- feat: Add flag to turn on or off creating/modifying CHANGELOG.md by @siketyan in https://github.com/Songmu/tagpr/pull/86

## [v0.1.3](https://github.com/Songmu/tagpr/compare/v0.1.2...v0.1.3) - 2022-09-05
- Fix a typo by @nghialv in https://github.com/Songmu/tagpr/pull/84

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
