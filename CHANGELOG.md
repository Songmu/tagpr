# Changelog

## [v0.0.12](https://github.com/Songmu/rcpr/compare/v0.0.11...v0.0.12) - 2022-08-24
- adjust default pull request body by @Songmu in https://github.com/Songmu/rcpr/pull/52
- fix remote name detection by @Songmu in https://github.com/Songmu/rcpr/pull/54

## [v0.0.11](https://github.com/Songmu/rcpr/compare/v0.0.10...v0.0.11) - 2022-08-21
- fix config key to rcpr.vPrefix from rcpr.v-prefix by @Songmu in https://github.com/Songmu/rcpr/pull/50
- reload version file after cherry-picking process by @Songmu in https://github.com/Songmu/rcpr/pull/51

## [v0.0.10](https://github.com/Songmu/rcpr/compare/v0.0.9...v0.0.10) - 2022-08-20
- config rcpr.v-prefix by @Songmu in https://github.com/Songmu/rcpr/pull/45

## [v0.0.9](https://github.com/Songmu/rcpr/compare/v0.0.8...v0.0.9) - 2022-08-20
- implement configuration file by @Songmu in https://github.com/Songmu/rcpr/pull/41
- skip version file detection with document files by @Songmu in https://github.com/Songmu/rcpr/pull/43

## [v0.0.8](https://github.com/Songmu/rcpr/compare/v0.0.7...v0.0.8) - 2022-08-18
- refine version file detection by @Songmu in https://github.com/Songmu/rcpr/pull/39

## [v0.0.7](https://github.com/Songmu/rcpr/compare/v0.0.6...v0.0.7) - 2022-08-18
- fix process around date when updating changelog by @Songmu in https://github.com/Songmu/rcpr/pull/37

## [v0.0.6](https://github.com/Songmu/rcpr/compare/v0.0.5...v0.0.6) - 2022-08-18
- create as many changelogs as possible if missing by @Songmu in https://github.com/Songmu/rcpr/pull/35

## [v0.0.5](https://github.com/Songmu/rcpr/compare/v0.0.4...v0.0.5) - 2022-08-17
- use fi.Mode().IsRegular() by @Songmu in https://github.com/Songmu/rcpr/pull/32
- implement updating CHANGELOG.md process by @Songmu in https://github.com/Songmu/rcpr/pull/34

## [v0.0.4](https://github.com/Songmu/rcpr/compare/v0.0.3...v0.0.4) - 2022-08-17
- create a normal release instead of a draft when tagging by @Songmu in https://github.com/Songmu/rcpr/pull/30

## [v0.0.3](https://github.com/Songmu/rcpr/compare/v0.0.2...v0.0.3) - 2022-08-17
- use personal access token in rcpr by @Songmu in https://github.com/Songmu/rcpr/pull/28
- introduce softprops/aciton-gh-release by @Songmu in https://github.com/Songmu/rcpr/pull/29

## [v0.0.2](https://github.com/Songmu/rcpr/compare/v0.0.1...v0.0.2) - 2022-08-17
- guess the next version from the label name convention by @Songmu in https://github.com/Songmu/rcpr/pull/20
- unshallow in initialize by @Songmu in https://github.com/Songmu/rcpr/pull/22
- retrieve next version after pushing changes into rc branch by @Songmu in https://github.com/Songmu/rcpr/pull/24
- introduce semv struct for representing semver by @Songmu in https://github.com/Songmu/rcpr/pull/25
- create a draft release at the same time it tags by @Songmu in https://github.com/Songmu/rcpr/pull/26

## [v0.0.1](https://github.com/Songmu/rcpr/commits/v0.0.1) - 2022-08-17
- add github.go for github client by @Songmu in https://github.com/Songmu/rcpr/pull/1
- create rc pull request when the default branch proceeded by @Songmu in https://github.com/Songmu/rcpr/pull/2
- dogfooding by @Songmu in https://github.com/Songmu/rcpr/pull/3
- set label to the pull request by @Songmu in https://github.com/Songmu/rcpr/pull/5
- change rc branch naming convention by @Songmu in https://github.com/Songmu/rcpr/pull/6
- adjust auto commit message by @Songmu in https://github.com/Songmu/rcpr/pull/8
- apply the commits added on the RC branch with cherry-pick by @Songmu in https://github.com/Songmu/rcpr/pull/9
- unshallow if a shallow repository by @Songmu in https://github.com/Songmu/rcpr/pull/10
- fix git log by @Songmu in https://github.com/Songmu/rcpr/pull/11
- parse git URL more precise by @Songmu in https://github.com/Songmu/rcpr/pull/12
- fix parseGitURL by @Songmu in https://github.com/Songmu/rcpr/pull/13
- refactor git.go by @Songmu in https://github.com/Songmu/rcpr/pull/14
- set user.email and user.name only if they aren't set by @Songmu in https://github.com/Songmu/rcpr/pull/15
- fix api base handling by @Songmu in https://github.com/Songmu/rcpr/pull/16
- take care of v-prefix or not in tags by @Songmu in https://github.com/Songmu/rcpr/pull/17
- Detect version file and update by @Songmu in https://github.com/Songmu/rcpr/pull/18
- tagging semver to merged rcpr by @Songmu in https://github.com/Songmu/rcpr/pull/19
