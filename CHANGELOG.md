# Changelog

## [v1.16.0](https://github.com/Songmu/tagpr/compare/v1.15.0...v1.16.0) - 2026-02-14
- docs: improve README for labels and env vars by @tokuhirom in https://github.com/Songmu/tagpr/pull/300
- Use scoped release yaml path in github releases by @wreulicke in https://github.com/Songmu/tagpr/pull/304
- fix: latestSemverTag returns empty for zero-padded calver tags by @k1LoW in https://github.com/Songmu/tagpr/pull/303
- build(deps): bump golang.org/x/oauth2 from 0.34.0 to 0.35.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/305
- build(deps): bump github.com/Songmu/gitconfig from 0.2.1 to 0.2.2 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/299
- Migrate gh2changelog to monorepo as local submodule by @Copilot in https://github.com/Songmu/tagpr/pull/307
- Fix: validate tagPrefix in isTagPR for monorepo isolation by @Copilot in https://github.com/Songmu/tagpr/pull/311

## [v1.15.0](https://github.com/Songmu/tagpr/compare/v1.14.0...v1.15.0) - 2026-02-01
- feat: allow `tagpr.calendarVersioning` to accept format string directly by @k1LoW in https://github.com/Songmu/tagpr/pull/295

## [v1.14.0](https://github.com/Songmu/tagpr/compare/v1.13.0...v1.14.0) - 2026-01-29
- feat: scoped release configuration by @wreulicke in https://github.com/Songmu/tagpr/pull/291
- [fix] care nil ReleaseYAML by @Songmu in https://github.com/Songmu/tagpr/pull/293

## [v1.13.0](https://github.com/Songmu/tagpr/compare/v1.12.1...v1.13.0) - 2026-01-29
- Add Calendar Versioning (CalVer) support by @fujiwara in https://github.com/Songmu/tagpr/pull/288
- build(deps): bump actions/checkout from 6.0.1 to 6.0.2 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/289

## [v1.12.1](https://github.com/Songmu/tagpr/compare/v1.12.0...v1.12.1) - 2026-01-21
- fix: pass changelog file path to gh2changelog by @wreulicke in https://github.com/Songmu/tagpr/pull/286

## [v1.12.0](https://github.com/Songmu/tagpr/compare/v1.11.1...v1.12.0) - 2026-01-21
- feat: add changelogFile for monorepo support by @wreulicke in https://github.com/Songmu/tagpr/pull/283
- build(deps): bump actions/setup-go from 6.1.0 to 6.2.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/281

## [v1.11.1](https://github.com/Songmu/tagpr/compare/v1.11.0...v1.11.1) - 2026-01-13
- fix: add check for both release.yml and release.yaml files by @nnnkkk7 in https://github.com/Songmu/tagpr/pull/275
- feat: add base_tag output for GitHub Actions by @178inaba in https://github.com/Songmu/tagpr/pull/277
- fix: scope git log to tagPrefix path in monorepo mode by @biosugar0 in https://github.com/Songmu/tagpr/pull/274

## [v1.11.0](https://github.com/Songmu/tagpr/compare/v1.10.0...v1.11.0) - 2026-01-06
- build(deps): bump golang.org/x/oauth2 from 0.33.0 to 0.34.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/265
- build(deps): bump codecov/codecov-action from 5.5.1 to 5.5.2 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/266
- build(deps): bump github.com/Songmu/gitsemvers from 0.0.3 to 0.1.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/269
- Add tagPrefix configuration for monorepo support by @biosugar0 in https://github.com/Songmu/tagpr/pull/268
- build(deps): bump github.com/Songmu/gh2changelog from 0.3.0 to 0.4.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/270

## [v1.10.0](https://github.com/Songmu/tagpr/compare/v1.9.2...v1.10.0) - 2025-12-14
- Preserve file modes by @hekki in https://github.com/Songmu/tagpr/pull/263

## [v1.9.2](https://github.com/Songmu/tagpr/compare/v1.9.1...v1.9.2) - 2025-12-13
- config: introduce setFromGitconfig, reloadField, reloadBoolField meth… by @12ya in https://github.com/Songmu/tagpr/pull/251

## [v1.9.1](https://github.com/Songmu/tagpr/compare/v1.9.0...v1.9.1) - 2025-12-13
- build(deps): bump actions/create-github-app-token from 2.1.1 to 2.1.4 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/244
- build(deps): bump haya14busa/action-update-semver from 1.5.0 to 1.5.1 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/245
- build(deps): bump golang.org/x/oauth2 from 0.31.0 to 0.32.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/247
- tagpr: Refactor label addition logic in Run method by @12ya in https://github.com/Songmu/tagpr/pull/252
- util.go: remove unnecessary middle man by @12ya in https://github.com/Songmu/tagpr/pull/250
- build(deps): bump golang.org/x/oauth2 from 0.32.0 to 0.33.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/256
- build(deps): bump actions/create-github-app-token from 2.1.4 to 2.2.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/259
- build(deps): bump actions/setup-go from 6.0.0 to 6.1.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/258
- build(deps): bump actions/checkout from 5.0.0 to 6.0.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/257
- build(deps): bump actions/create-github-app-token from 2.2.0 to 2.2.1 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/261
- build(deps): bump actions/checkout from 6.0.0 to 6.0.1 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/260
- tagpr: Refactor version file and command execution logic by @12ya in https://github.com/Songmu/tagpr/pull/253
- tagpr: add *tag pr method getNextLabels and debloat Run func by @12ya in https://github.com/Songmu/tagpr/pull/249

## [v1.9.0](https://github.com/Songmu/tagpr/compare/v1.8.4...v1.9.0) - 2025-09-08
- build(deps): bump haya14busa/action-update-semver from 1.3.0 to 1.5.0 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/240
- chore: replace github.Bool to github.Str by @Songmu in https://github.com/Songmu/tagpr/pull/241
- feat: replace compare link in pull request body by @Songmu in https://github.com/Songmu/tagpr/pull/243

## [v1.8.4](https://github.com/Songmu/tagpr/compare/v1.8.3...v1.8.4) - 2025-09-08
- fix: specify releaseBranch when creating changelog by @Songmu in https://github.com/Songmu/tagpr/pull/238

## [v1.8.3](https://github.com/Songmu/tagpr/compare/v1.8.2...v1.8.3) - 2025-09-08
- disable cache on release by @Songmu in https://github.com/Songmu/tagpr/pull/236

## [v1.8.2](https://github.com/Songmu/tagpr/compare/v1.8.1...v1.8.2) - 2025-09-08
- fix: set token to git remote show origin by @Songmu in https://github.com/Songmu/tagpr/pull/234

## [v1.8.1](https://github.com/Songmu/tagpr/compare/v1.8.0...v1.8.1) - 2025-09-08
- update go-github from v66 to v74 by @Songmu in https://github.com/Songmu/tagpr/pull/229
- fix: check exsisting extraheader before set token by @Songmu in https://github.com/Songmu/tagpr/pull/232

## [v1.8.0](https://github.com/Songmu/tagpr/compare/v1.7.2...v1.8.0) - 2025-09-07
- set proper settings for github actions by @Songmu in https://github.com/Songmu/tagpr/pull/223
- feat: using a token for remote access with git commands by @Songmu in https://github.com/Songmu/tagpr/pull/225
- doc: support persist-credentials: false on checkout by @Songmu in https://github.com/Songmu/tagpr/pull/226
- fix: care GitHub Enterprise Cloud by @Songmu in https://github.com/Songmu/tagpr/pull/227
- chore: use new syntaxes by @Songmu in https://github.com/Songmu/tagpr/pull/228

## [v1.7.2](https://github.com/Songmu/tagpr/compare/v1.7.1...v1.7.2) - 2025-09-05
- introduce immutable action with ghr by @Songmu in https://github.com/Songmu/tagpr/pull/221

## [v1.7.1](https://github.com/Songmu/tagpr/compare/v1.7.0...v1.7.1) - 2025-09-05
- chore: to introduce immutable action adjust actions by @Songmu in https://github.com/Songmu/tagpr/pull/213
- update go version and deps by @Songmu in https://github.com/Songmu/tagpr/pull/220
- build(deps): bump actions/create-github-app-token from 1.12.0 to 2.1.1 by @dependabot[bot] in https://github.com/Songmu/tagpr/pull/215

## [v1.7.0](https://github.com/Songmu/tagpr/compare/v1.6.1...v1.7.0) - 2025-06-14
- Improve the composite action security by @5ouma in https://github.com/Songmu/tagpr/pull/207
- tagpr.go: cleanup by @12ya in https://github.com/Songmu/tagpr/pull/209
- feat: add-post-version-command by @kiyo-matsu in https://github.com/Songmu/tagpr/pull/212

## [v1.6.1](https://github.com/Songmu/tagpr/compare/v1.6.0...v1.6.1) - 2025-05-15
- using action version for tagpr installer of action by @kenchan0130 in https://github.com/Songmu/tagpr/pull/204
- udpate version 2 times in action.yml by @Songmu in https://github.com/Songmu/tagpr/pull/206

## [v1.6.0](https://github.com/Songmu/tagpr/compare/v1.5.2...v1.6.0) - 2025-05-15
- Reducing supply chain risk at actions by @kenchan0130 in https://github.com/Songmu/tagpr/pull/201
- Add docs about persist-credentials of actions/checkout by @kenchan0130 in https://github.com/Songmu/tagpr/pull/200
- use versioned install.sh with variables in action.yml by @Songmu in https://github.com/Songmu/tagpr/pull/203
- Support signed commit by @yasu89 in https://github.com/Songmu/tagpr/pull/199

## [v1.5.2](https://github.com/Songmu/tagpr/compare/v1.5.1...v1.5.2) - 2025-04-18
- Fix 403 error by adding 'issues: write' permission to GitHub Actions. by @monochromegane in https://github.com/Songmu/tagpr/pull/194
- resolves #197 by @12ya in https://github.com/Songmu/tagpr/pull/198

## [v1.5.1](https://github.com/Songmu/tagpr/compare/v1.5.0...v1.5.1) - 2025-01-07
- versionfile.go: omit .github directory from processing by @fujiwara in https://github.com/Songmu/tagpr/pull/191
- Use `releaseBranch` variable instead of the hard-coded "main" in `git log` command by @mmizutani in https://github.com/Songmu/tagpr/pull/190

## [v1.5.0](https://github.com/Songmu/tagpr/compare/v1.4.3...v1.5.0) - 2024-10-27
- Get the config file path from the environment variable by @5ouma in https://github.com/Songmu/tagpr/pull/186
- Specify a template text directly in the config file by @5ouma in https://github.com/Songmu/tagpr/pull/187
- Change the commit message prefix by @5ouma in https://github.com/Songmu/tagpr/pull/188

## [v1.4.3](https://github.com/Songmu/tagpr/compare/v1.4.2...v1.4.3) - 2024-10-22
- update deps and Go version by @Songmu in https://github.com/Songmu/tagpr/pull/184

## [v1.4.2](https://github.com/Songmu/tagpr/compare/v1.4.1...v1.4.2) - 2024-09-21
- Update README. by @monochromegane in https://github.com/Songmu/tagpr/pull/181
- Retry when "Secondary rate limit" error occurs in GitHub API by @snaka in https://github.com/Songmu/tagpr/pull/183

## [v1.4.1](https://github.com/Songmu/tagpr/compare/v1.4.0...v1.4.1) - 2024-09-09
- use static option to build tagpr binaries by @vvakame in https://github.com/Songmu/tagpr/pull/179

## [v1.4.0](https://github.com/Songmu/tagpr/compare/v1.3.0...v1.4.0) - 2024-08-11
- fix typo by @mocyuto in https://github.com/Songmu/tagpr/pull/175
- fix: Unstable search issue behavior by @snaka in https://github.com/Songmu/tagpr/pull/178

## [v1.3.0](https://github.com/Songmu/tagpr/compare/v1.2.0...v1.3.0) - 2024-05-15
- fix: Typo in README by @tgeorg-ethz in https://github.com/Songmu/tagpr/pull/172
- Add showGHError() by @fujiwara in https://github.com/Songmu/tagpr/pull/173

## [v1.2.0](https://github.com/Songmu/tagpr/compare/v1.1.4...v1.2.0) - 2023-12-31
- update: added configuration of Github Enteprise by @ponkio-o in https://github.com/Songmu/tagpr/pull/162
- Refer to the next version with command by @k2tzumi in https://github.com/Songmu/tagpr/pull/165
- update deps by @Songmu in https://github.com/Songmu/tagpr/pull/166

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
