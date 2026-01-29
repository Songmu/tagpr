package tagpr

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Songmu/gitconfig"
	"github.com/google/go-github/v82/github"
)

const (
	defaultConfigFile    = ".tagpr"
	defaultConfigContent = `# config file for the tagpr in git config format
# The tagpr generates the initial configuration, which you can rewrite to suit your environment.
# CONFIGURATIONS:
#   tagpr.releaseBranch
#       Generally, it is "main." It is the branch for releases. The tagpr tracks this branch,
#       creates or updates a pull request as a release candidate, or tags when they are merged.
#
#   tagpr.versionFile
#       Versioning file containing the semantic version needed to be updated at release.
#       It will be synchronized with the "git tag".
#       Often this is a meta-information file such as gemspec, setup.cfg, package.json, etc.
#       Sometimes the source code file, such as version.go or Bar.pm, is used.
#       If you do not want to use versioning files but only git tags, specify the "-" string here.
#       You can specify multiple version files by comma separated strings.
#
#   tagpr.vPrefix
#       Flag whether or not v-prefix is added to semver when git tagging. (e.g. v1.2.3 if true)
#       This is only a tagging convention, not how it is described in the version file.
#
#   tagpr.changelog (Optional)
#       Flag whether or not changelog is added or changed during the release.
#
#   tagpr.command (Optional)
#       Command to change files just before release and versioning.
#
#   tagpr.postVersionCommand (Optional)
#       Command to change files just after versioning.
#
#   tagpr.template (Optional)
#       Pull request template file in go template format
#
#   tagpr.templateText (Optional)
#       Pull request template text in go template format
#
#   tagpr.release (Optional)
#       GitHub Release creation behavior after tagging [true, draft, false]
#       If this value is not set, the release is to be created.
#
#   tagpr.majorLabels (Optional)
#       Label of major update targets. Default is [major]
#
#   tagpr.minorLabels (Optional)
#       Label of minor update targets. Default is [minor]
#
#   tagpr.commitPrefix (Optional)
#       Prefix of commit message. Default is "[tagpr]"
#
#   tagpr.tagPrefix (Optional)
#       Tag prefix for monorepo support. This allows managing multiple packages
#       with independent versioning in a single repository.
#       The prefix is prepended to the version tag with a slash separator.
#       Trailing slashes in the prefix are handled automatically.
#       Examples:
#         - "tools" produces tags like "tools/v1.2.3"
#         - "backend/api" produces tags like "backend/api/v1.0.0"
#
#   tagpr.calendarVersioning (Optional)
#       Use Calendar Versioning instead of Semantic Versioning.
#       Must be explicitly set to true to enable. Default is false (Semantic Versioning).
#
#   tagpr.calendarVersioningFormat (Optional)
#       Calendar Versioning format string. Only used when calendarVersioning is true.
#       Default is "YYYY.MMDD.MICRO".
#       Available tokens (see https://calver.org):
#         Year:    YYYY (4-digit), YY (2-digit), 0Y (zero-padded 2-digit)
#         Month:   MM (no padding), 0M (zero-padded)
#         Week:    WW (no padding), 0W (zero-padded)
#         Day:     DD (no padding), 0D (zero-padded)
#         Micro:   MICRO (auto-incrementing patch number for same date)
#       Examples:
#         - "YYYY.MMDD.MICRO" -> 2026.123.0 (Jan 23)
#         - "YYYY.0M.MICRO" -> 2026.01.0
#         - "YY.0M0D.MICRO" -> 26.0123.0
#
[tagpr]
`
	defaultMajorLabels              = "major"
	defaultMinorLabels              = "minor"
	defaultCommitPrefix             = "[tagpr]"
	defaultCalendarVersioningFormat = "YYYY.MMDD.MICRO"
	envConfigFile                   = "TAGPR_CONFIG_FILE"
	envReleaseBranch                = "TAGPR_RELEASE_BRANCH"
	envVersionFile                  = "TAGPR_VERSION_FILE"
	envVPrefix                      = "TAGPR_VPREFIX"
	envChangelog                    = "TAGPR_CHANGELOG"
	envCommand                      = "TAGPR_COMMAND"
	envPostVersionCommand           = "TAGPR_POST_VERSION_COMMAND"
	envTemplate                     = "TAGPR_TEMPLATE"
	envTemplateText                 = "TAGPR_TEMPLATE_TEXT"
	envRelease                      = "TAGPR_RELEASE"
	envMajorLabels                  = "TAGPR_MAJOR_LABELS"
	envMinorLabels                  = "TAGPR_MINOR_LABELS"
	envCommitPrefix                 = "TAGPR_COMMIT_PREFIX"
	envTagPrefix                    = "TAGPR_TAG_PREFIX"
	envChangelogFile                = "TAGPR_CHANGELOG_FILE"
	envCalendarVersioning           = "TAGPR_CALENDAR_VERSIONING"
	envCalendarVersioningFormat     = "TAGPR_CALENDAR_VERSIONING_FORMAT"
	envReleaseYAMLPath              = "TAGPR_RELEASE_YAML_PATH"
	configReleaseBranch             = "tagpr.releaseBranch"
	configVersionFile               = "tagpr.versionFile"
	configVPrefix                   = "tagpr.vPrefix"
	configChangelog                 = "tagpr.changelog"
	configCommand                   = "tagpr.command"
	configPostVersionCommand        = "tagpr.postVersionCommand"
	configTemplate                  = "tagpr.template"
	configTemplateText              = "tagpr.templateText"
	configRelease                   = "tagpr.release"
	configMajorLabels               = "tagpr.majorLabels"
	configMinorLabels               = "tagpr.minorLabels"
	configCommitPrefix              = "tagpr.commitPrefix"
	configTagPrefix                 = "tagpr.tagPrefix"
	configChangelogFile             = "tagpr.changelogFile"
	configCalendarVersioning        = "tagpr.calendarVersioning"
	configCalendarVersioningFormat  = "tagpr.calendarVersioningFormat"
	configReleaseYAMLPath           = "tagpr.releaseYAMLPath"
)

type config struct {
	releaseBranch            *string
	versionFile              *string
	command                  *string
	postVersionCommand       *string
	template                 *string
	templateText             *string
	release                  *string
	vPrefix                  *bool
	changelog                *bool
	majorLabels              *string
	minorLabels              *string
	commitPrefix             *string
	tagPrefix                *string
	changelogFile            *string
	calendarVersioning       *bool
	calendarVersioningFormat *string
	releaseYamlPath          *string

	conf      string
	gitconfig *gitconfig.Config
}

func newConfig(gitPath string) (*config, error) {
	var conf = defaultConfigFile
	if cf := os.Getenv(envConfigFile); cf != "" {
		conf = cf
	}
	cfg := &config{
		conf:      conf,
		gitconfig: &gitconfig.Config{GitPath: gitPath, File: conf},
	}
	err := cfg.Reload()
	return cfg, err
}

func (cfg *config) Reload() error {
	cfg.reloadField(&cfg.releaseBranch, configReleaseBranch, envReleaseBranch, "")

	cfg.reloadField(&cfg.versionFile, configVersionFile, envVersionFile, "")

	if err := cfg.reloadBoolField(&cfg.vPrefix, envVPrefix, configVPrefix); err != nil {
		return err
	}

	if err := cfg.reloadBoolField(&cfg.changelog, envChangelog, configChangelog); err != nil {
		return err
	}

	cfg.reloadField(&cfg.command, configCommand, envCommand, "")

	cfg.reloadField(&cfg.postVersionCommand, configPostVersionCommand, envPostVersionCommand, "")

	cfg.reloadField(&cfg.template, configTemplate, envTemplate, "")

	cfg.reloadField(&cfg.templateText, configTemplateText, envTemplateText, "")

	cfg.reloadField(&cfg.release, configRelease, envRelease, "")

	cfg.reloadField(&cfg.majorLabels, configMajorLabels, envMajorLabels, defaultMajorLabels)

	cfg.reloadField(&cfg.minorLabels, configMinorLabels, envMinorLabels, defaultMinorLabels)

	cfg.reloadField(&cfg.commitPrefix, configCommitPrefix, envCommitPrefix, defaultCommitPrefix)

	cfg.reloadField(&cfg.tagPrefix, configTagPrefix, envTagPrefix, "")

	cfg.reloadField(&cfg.changelogFile, configChangelogFile, envChangelogFile, "")

	cfg.reloadField(&cfg.releaseYamlPath, configReleaseYAMLPath, envReleaseYAMLPath, "")

	if err := cfg.reloadBoolField(&cfg.calendarVersioning, envCalendarVersioning, configCalendarVersioning); err != nil {
		return err
	}

	cfg.reloadField(&cfg.calendarVersioningFormat, configCalendarVersioningFormat, envCalendarVersioningFormat, defaultCalendarVersioningFormat)

	if err := validateCalendarVersioningFormat(cfg.CalendarVersioningFormat()); err != nil {
		return err
	}

	return nil
}

func (cfg *config) setFromGitconfig(dst **string, gitconfigSrc, defaultSrc string) {
	if val, err := cfg.gitconfig.Get(gitconfigSrc); err == nil {
		*dst = github.Ptr(val)
	} else {
		if defaultSrc != "" {
			*dst = github.Ptr(defaultSrc)
		}
	}
}

func (cfg *config) reloadField(dst **string, gitconfigSrc, envVal, defaultSrc string) {
	if val := os.Getenv(envVal); val != "" {
		*dst = github.Ptr(val)
	} else {
		cfg.setFromGitconfig(dst, gitconfigSrc, defaultSrc)
	}
}

func (cfg *config) reloadBoolField(dst **bool, envVal, gitconfigSrc string) error {
	if val := os.Getenv(envVal); val != "" {
		if b, err := strconv.ParseBool(val); err != nil {
			return err
		} else {
			*dst = github.Ptr(b)
		}
	} else {
		if b, err := cfg.gitconfig.Bool(gitconfigSrc); err == nil {
			*dst = github.Ptr(b)
		}
	}

	return nil
}

func (cfg *config) set(key, value string) error {
	if !exists(cfg.conf) {
		if err := cfg.initializeFile(); err != nil {
			return err
		}
	}
	if value == "" {
		value = "-" // value "-" represents null (really?)
	}
	_, err := cfg.gitconfig.Do(key, value)
	if err != nil {
		// in this case, config file might be invalid or broken, so retry once.
		if err = cfg.initializeFile(); err != nil {
			return err
		}
		_, err = cfg.gitconfig.Do(key, value)
	}
	return err
}

func (cfg *config) initializeFile() error {
	if err := os.RemoveAll(cfg.conf); err != nil {
		return err
	}
	if err := os.WriteFile(cfg.conf, []byte(defaultConfigContent), 0644); err != nil {
		return err
	}
	return nil
}

func (cfg *config) SetReleaseBranch(br string) error {
	if err := cfg.set(configReleaseBranch, br); err != nil {
		return err
	}
	cfg.releaseBranch = github.Ptr(br)
	return nil
}

func (cfg *config) SetVersionFile(fpath string) error {
	if err := cfg.set(configVersionFile, fpath); err != nil {
		return err
	}
	cfg.versionFile = github.Ptr(fpath)
	return nil
}

func (cfg *config) SetVPrefix(vPrefix bool) error {
	if err := cfg.set(configVPrefix, strconv.FormatBool(vPrefix)); err != nil {
		return err
	}
	cfg.vPrefix = github.Ptr(vPrefix)
	return nil
}

func stringify(pstr *string) string {
	if pstr == nil || *pstr == "-" {
		return ""
	}
	return *pstr
}

func (cfg *config) ReleaseBranch() string {
	return stringify(cfg.releaseBranch)
}

func (cfg *config) VersionFile() string {
	if cfg.versionFile == nil {
		return ""
	}
	return *cfg.versionFile
}

func (cfg *config) Command() string {
	return stringify(cfg.command)
}

func (cfg *config) PostVersionCommand() string {
	return stringify(cfg.postVersionCommand)
}

func (cfg *config) Template() string {
	return stringify(cfg.template)
}

func (cfg *config) TemplateText() string {
	return stringify(cfg.templateText)
}

func (cfg *config) Release() bool {
	rel := strings.ToLower(stringify(cfg.release))
	if rel == "draft" || rel == "" {
		return true
	}
	b, err := strconv.ParseBool(rel)
	if err != nil {
		return true
	}
	return b
}

func (cfg *config) ReleaseDraft() bool {
	return strings.ToLower(stringify(cfg.release)) == "draft"
}

func (cfg *config) MajorLabels() []string {
	labels := strings.Split(stringify(cfg.majorLabels), ",")

	for i, v := range labels {
		labels[i] = strings.TrimSpace(v)
	}

	return labels
}

func (cfg *config) MinorLabels() []string {
	labels := strings.Split(stringify(cfg.minorLabels), ",")

	for i, v := range labels {
		labels[i] = strings.TrimSpace(v)
	}

	return labels
}

func (cfg *config) CommitPrefix() string {
	return stringify(cfg.commitPrefix)
}

func (cfg *config) TagPrefix() string {
	return stringify(cfg.tagPrefix)
}

func (cfg *config) ChangelogFile() string {
	if cfg.changelogFile == nil {
		return "CHANGELOG.md"
	}
	return stringify(cfg.changelogFile)
}

func (cfg *config) CalendarVersioning() bool {
	if cfg.calendarVersioning == nil {
		return false
	}
	return *cfg.calendarVersioning
}

func (cfg *config) SetCalendarVersioning(calVer bool) error {
	if err := cfg.set(configCalendarVersioning, strconv.FormatBool(calVer)); err != nil {
		return err
	}
	cfg.calendarVersioning = github.Ptr(calVer)
	return nil
}

func (cfg *config) CalendarVersioningFormat() string {
	return stringify(cfg.calendarVersioningFormat)
}

func (cfg *config) SetCalendarVersioningFormat(format string) error {
	if err := validateCalendarVersioningFormat(format); err != nil {
		return err
	}
	if err := cfg.set(configCalendarVersioningFormat, format); err != nil {
		return err
	}
	cfg.calendarVersioningFormat = github.Ptr(format)
	return nil
}

func validateCalendarVersioningFormat(format string) error {
	if strings.Contains(format, "MAJOR") {
		return fmt.Errorf("MAJOR token is not allowed in calendarVersioningFormat: CalVer uses date-based versioning and ignores major/minor labels")
	}
	if strings.Contains(format, "MINOR") {
		return fmt.Errorf("MINOR token is not allowed in calendarVersioningFormat: CalVer uses date-based versioning and ignores major/minor labels")
	}
	return nil
}

func (cfg *config) ReleaseYAMLPath() string {
	return stringify(cfg.releaseYamlPath)
}
