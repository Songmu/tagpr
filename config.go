package rcpr

import (
	"os"
)

const (
	defaultConfigFile    = ".rcpr"
	defaultConfigContent = `# config file for rcpr in git config format
[rcpr]
`
	envReleaseBranch    = "RCPR_RELEASE_BRANCH"
	envVersionFile      = "RCPR_VERSION_FILE"
	configReleaseBranch = "rcpr.releaseBranch"
	configVersionFile   = "rcpr.versionFile"
)

type config struct {
	releaseBranch *configValue
	versionFile   *configValue

	c    *commander
	conf string
}

func newConfig(c *commander) *config {
	cfg := &config{conf: defaultConfigFile, c: c}
	if rb := os.Getenv(envReleaseBranch); rb != "" {
		cfg.releaseBranch = &configValue{
			value:  rb,
			source: srcEnv,
		}
	} else {
		out, _, err := c.gitE("config", "-f", cfg.conf, configReleaseBranch)
		if err != nil {
			cfg.releaseBranch = &configValue{
				value:  out,
				source: srcConfigFile,
			}
		}
	}

	if rb := os.Getenv(envVersionFile); rb != "" {
		cfg.releaseBranch = &configValue{
			value:  rb,
			source: srcEnv,
		}
	} else {
		out, _, err := c.gitE("config", "-f", cfg.conf, configVersionFile)
		if err != nil {
			cfg.releaseBranch = &configValue{
				value:  out,
				source: srcConfigFile,
			}
		}
	}
	return cfg
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
	_, _, err := cfg.c.gitE("config", "-f", cfg.conf, key, value)
	if err != nil {
		// in this case, config file might be invalid or broken, so retry once.
		if err = cfg.initializeFile(); err != nil {
			return err
		}
		_, _, err = cfg.c.gitE("config", "-f", cfg.conf, key, value)
	}
	return err
}

func (cfg *config) initializeFile() error {
	if err := os.RemoveAll(cfg.conf); err != nil {
		return err
	}
	if err := os.WriteFile(cfg.conf, []byte(defaultConfigContent), 0666); err != nil {
		return err
	}
	return nil
}

func (cfg *config) SetRelaseBranch(br string) error {
	if err := cfg.set(configReleaseBranch, br); err != nil {
		return err
	}
	cfg.releaseBranch = &configValue{
		value:  br,
		source: srcDetect,
	}
	return nil
}

func (cfg *config) SetVersionFile(fpath string) error {
	if err := cfg.set(configVersionFile, fpath); err != nil {
		return err
	}
	cfg.versionFile = &configValue{
		value:  fpath,
		source: srcDetect,
	}
	return nil
}

func (cfg *config) RelaseBranch() *configValue {
	return cfg.releaseBranch
}

func (cfg *config) VersionFile() *configValue {
	return cfg.versionFile
}

type configValue struct {
	value  string
	source configSource
}

func (cv *configValue) String() string {
	if cv.value == "-" {
		return ""
	}
	return cv.value
}

func (cv *configValue) Empty() bool {
	return cv.String() == ""
}

type configSource int

const (
	srcEnv configSource = iota
	srcConfigFile
	srcDetect
)
