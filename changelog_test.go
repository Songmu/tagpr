package rcpr

import (
	"testing"
	"time"
)

func TestConvertKeepAChangelogFormat(t *testing.T) {

	input := `## What's Changed
* add github.go for github client by @Songmu in https://github.com/Songmu/rcpr/pull/1
* create rc pull request when the default branch proceeded by @Songmu in https://github.com/Songmu/rcpr/pull/2
* dogfooding by @Songmu in https://github.com/Songmu/rcpr/pull/3
* set label to the pull request by @Songmu in https://github.com/Songmu/rcpr/pull/5
* change rc branch naming convention by @Songmu in https://github.com/Songmu/rcpr/pull/6
* adjust auto commit message by @Songmu in https://github.com/Songmu/rcpr/pull/8
* apply the commits added on the RC branch with cherry-pick by @Songmu in https://github.com/Songmu/rcpr/pull/9
* unshallow if a shallow repository by @Songmu in https://github.com/Songmu/rcpr/pull/10
* fix git log by @Songmu in https://github.com/Songmu/rcpr/pull/11
* parse git URL more precise by @Songmu in https://github.com/Songmu/rcpr/pull/12
* fix parseGitURL by @Songmu in https://github.com/Songmu/rcpr/pull/13
* refactor git.go by @Songmu in https://github.com/Songmu/rcpr/pull/14
* set user.email and user.name only if they aren't set by @Songmu in https://github.com/Songmu/rcpr/pull/15
* fix api base handling by @Songmu in https://github.com/Songmu/rcpr/pull/16
* take care of v-prefix or not in tags by @Songmu in https://github.com/Songmu/rcpr/pull/17
* Detect version file and update by @Songmu in https://github.com/Songmu/rcpr/pull/18
* tagging semver to merged rcpr by @Songmu in https://github.com/Songmu/rcpr/pull/19

## New Contributors
* @Songmu made their first contribution in https://github.com/Songmu/rcpr/pull/1

**Full Changelog**: https://github.com/Songmu/rcpr/commits/v0.0.1
`

	expect := `## [v0.0.1](https://github.com/Songmu/rcpr/commits/v0.0.1) - 2022-08-16
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
`

	got := convertKeepAChangelogFormat(input, time.Date(2022, time.August, 16, 18, 10, 10, 0, time.UTC))
	if got != expect {
		t.Errorf("error:\n %s", got)
	}
}
