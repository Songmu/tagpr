package tagpr

import (
	"os"
	"strconv"
	"strings"

	"github.com/Songmu/gitconfig"
	"github.com/google/go-github/v74/github"
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
[tagpr]
`
	defaultMajorLabels       = "major"
	defaultMinorLabels       = "minor"
	defaultCommitPrefix      = "[tagpr]"
	envConfigFile            = "TAGPR_CONFIG_FILE"
	envReleaseBranch         = "TAGPR_RELEASE_BRANCH"
	envVersionFile           = "TAGPR_VERSION_FILE"
	envVPrefix               = "TAGPR_VPREFIX"
	envChangelog             = "TAGPR_CHANGELOG"
	envCommand               = "TAGPR_COMMAND"
	envPostVersionCommand    = "TAGPR_POST_VERSION_COMMAND"
	envTemplate              = "TAGPR_TEMPLATE"
	envTemplateText          = "TAGPR_TEMPLATE_TEXT"
	envRelease               = "TAGPR_RELEASE"
	envMajorLabels           = "TAGPR_MAJOR_LABELS"
	envMinorLabels           = "TAGPR_MINOR_LABELS"
	envCommitPrefix          = "TAGPR_COMMIT_PREFIX"
	configReleaseBranch      = "tagpr.releaseBranch"
	configVersionFile        = "tagpr.versionFile"
	configVPrefix            = "tagpr.vPrefix"
	configChangelog          = "tagpr.changelog"
	configCommand            = "tagpr.command"
	configPostVersionCommand = "tagpr.postVersionCommand"
	configTemplate           = "tagpr.template"
	configTemplateText       = "tagpr.templateText"
	configRelease            = "tagpr.release"
	configMajorLabels        = "tagpr.majorLabels"
	configMinorLabels        = "tagpr.minorLabels"
	configCommitPrefix       = "tagpr.commitPrefix"
)

type config struct {
	releaseBranch      *string
	versionFile        *string
	command            *string
	postVersionCommand *string
	template           *string
	templateText       *string
	release            *string
	vPrefix            *bool
	changelog          *bool
	majorLabels        *string
	minorLabels        *string
	commitPrefix       *string

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
	if rb := os.Getenv(envReleaseBranch); rb != "" {
		cfg.releaseBranch = github.String(rb)
	} else {
		out, err := cfg.gitconfig.Get(configReleaseBranch)
		if err == nil {
			cfg.releaseBranch = github.String(out)
		}
	}

	if rb := os.Getenv(envVersionFile); rb != "" {
		cfg.versionFile = github.String(rb)
	} else {
		out, err := cfg.gitconfig.Get(configVersionFile)
		if err == nil {
			cfg.versionFile = github.String(out)
		}
	}

	if vPrefix := os.Getenv(envVPrefix); vPrefix != "" {
		b, err := strconv.ParseBool(vPrefix)
		if err != nil {
			return err
		}
		cfg.vPrefix = github.Bool(b)
	} else {
		b, err := cfg.gitconfig.Bool(configVPrefix)
		if err == nil {
			cfg.vPrefix = github.Bool(b)
		}
	}

	if changelog := os.Getenv(envChangelog); changelog != "" {
		b, err := strconv.ParseBool(changelog)
		if err != nil {
			return err
		}
		cfg.changelog = github.Bool(b)
	} else {
		b, err := cfg.gitconfig.Bool(configChangelog)
		if err == nil {
			cfg.changelog = github.Bool(b)
		}
	}

	if command := os.Getenv(envCommand); command != "" {
		cfg.command = github.String(command)
	} else {
		command, err := cfg.gitconfig.Get(configCommand)
		if err == nil {
			cfg.command = github.String(command)
		}
	}

	if postCommand := os.Getenv(envPostVersionCommand); postCommand != "" {
		cfg.postVersionCommand = github.String(postCommand)
	} else {
		postCommand, err := cfg.gitconfig.Get(configPostVersionCommand)
		if err == nil {
			cfg.postVersionCommand = github.String(postCommand)
		}
	}

	if tmpl := os.Getenv(envTemplate); tmpl != "" {
		cfg.template = github.String(tmpl)
	} else {
		tmpl, err := cfg.gitconfig.Get(configTemplate)
		if err == nil {
			cfg.template = github.String(tmpl)
		}
	}

	if tmplTxt := os.Getenv(envTemplateText); tmplTxt != "" {
		cfg.templateText = github.String(tmplTxt)
	} else {
		tmplTxt, err := cfg.gitconfig.Get(configTemplateText)
		if err == nil {
			cfg.templateText = github.String(tmplTxt)
		}
	}

	if rel := os.Getenv(envRelease); rel != "" {
		cfg.release = github.String(rel)
	} else {
		rel, err := cfg.gitconfig.Get(configRelease)
		if err == nil {
			cfg.release = github.String(rel)
		}
	}

	if major := os.Getenv(envMajorLabels); major != "" {
		cfg.majorLabels = github.String(major)
	} else {
		major, err := cfg.gitconfig.Get(configMajorLabels)
		if err == nil {
			cfg.majorLabels = github.String(major)
		} else {
			cfg.majorLabels = github.String(defaultMajorLabels)
		}
	}

	if minor := os.Getenv(envMinorLabels); minor != "" {
		cfg.minorLabels = github.String(minor)
	} else {
		minor, err := cfg.gitconfig.Get(configMinorLabels)
		if err == nil {
			cfg.minorLabels = github.String(minor)
		} else {
			cfg.minorLabels = github.String(defaultMinorLabels)
		}
	}

	if prefix := os.Getenv(envCommitPrefix); prefix != "" {
		cfg.commitPrefix = github.String(prefix)
	} else {
		prefix, err := cfg.gitconfig.Get(configCommitPrefix)
		if err == nil {
			cfg.commitPrefix = github.String(prefix)
		} else {
			cfg.commitPrefix = github.String(defaultCommitPrefix)
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
	cfg.releaseBranch = github.String(br)
	return nil
}

func (cfg *config) SetVersionFile(fpath string) error {
	if err := cfg.set(configVersionFile, fpath); err != nil {
		return err
	}
	cfg.versionFile = github.String(fpath)
	return nil
}

func (cfg *config) SetVPrefix(vPrefix bool) error {
	if err := cfg.set(configVPrefix, strconv.FormatBool(vPrefix)); err != nil {
		return err
	}
	cfg.vPrefix = github.Bool(vPrefix)
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
