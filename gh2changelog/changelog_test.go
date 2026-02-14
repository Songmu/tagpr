package gh2changelog

import (
	"testing"
	"time"
)

func TestConvertKeepAChangelogFormat(t *testing.T) {
	input := `<!-- Release notes generated using configuration in .github/release.yml at v0.0.12 -->

## What's Changed
* add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
* tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19

## New Contributors
* @Songmu made their first contribution in https://github.com/Songmu/gh2changelog/pull/1

**Full Changelog**: https://github.com/Songmu/gh2changelog/commits/v0.0.1
`

	expect := `## [v0.0.1](https://github.com/Songmu/gh2changelog/commits/v0.0.1) - 2022-08-16
- add github.go for github client by @Songmu in https://github.com/Songmu/gh2changelog/pull/1
- tagging semver to merged gh2changelog by @Songmu in https://github.com/Songmu/gh2changelog/pull/19
`

	ti := time.Date(2022, time.August, 16, 18, 10, 10, 0, time.UTC)
	got := convertKeepAChangelogFormat(input, ti)
	if got != expect {
		t.Errorf("error:\n %s", got)
	}

	input2 := `<!-- Release notes generated using configuration in .github/release.yml at v0.0.12 -->

**Full Changelog**: https://github.com/Songmu/godzil/compare/v0.20.12...v1.0.0`

	expect2 := "## [v1.0.0](https://github.com/Songmu/godzil/compare/v0.20.12...v1.0.0) - 2022-08-16\n"

	got2 := convertKeepAChangelogFormat(input2, ti)
	if got2 != expect2 {
		t.Errorf("error:\n %s", got2)
	}
}
