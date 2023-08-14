package tagpr

import (
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

type semv struct {
	v                *semver.Version
	formattedVersion *string

	vPrefix bool
}

func newSemver(v string, versionFormat *string) (*semv, error) {
	var err error
	sv := &semv{}
	if versionFormat != nil {
		currentTime := time.Now()
		formattedVersion := currentTime.Format(*versionFormat)
		sv.formattedVersion = &formattedVersion
		sv.vPrefix = formattedVersion[0] == 'v'
	} else {
		sv.v, err = semver.NewVersion(v)
		if err != nil {
			return nil, err
		}
		sv.vPrefix = v[0] == 'v'
	}
	return sv, nil
}

func (sv *semv) Naked() string {
	if sv.formattedVersion != nil {
		return *sv.formattedVersion
	}
	return sv.v.String()
}

func (sv *semv) Tag() string {
	if sv.vPrefix {
		return "v" + sv.Naked()
	}
	return sv.Naked()
}

func (sv *semv) GuessNext(labels []string, defaultVariable *string) *semv {
	if sv.formattedVersion != nil {
		for _, label := range labels {
			var separator = ""
			switch true {
			case strings.HasPrefix(label, autoLableName+":"):
				separator = ":"
			case strings.HasPrefix(label, autoLableName+"/"):
				separator = "/"
			}
			if separator != "" {
				variable := strings.TrimPrefix(label, autoLableName+separator)
				nextVersion := strings.ReplaceAll(*sv.formattedVersion, "${variable}", variable)
				return &semv{
					v:                sv.v,
					formattedVersion: &nextVersion,
					vPrefix:          sv.vPrefix,
				}
			}
		}
		return sv
	}
	var isMajor, isMinor bool
	for _, l := range labels {
		switch l {
		case autoLableName + ":major", autoLableName + "/major":
			isMajor = true
		case autoLableName + ":minor", autoLableName + "/minor":
			isMinor = true
		}
	}

	var nextv semver.Version
	switch {
	case isMajor:
		nextv = sv.v.IncMajor()
	case isMinor:
		nextv = sv.v.IncMinor()
	default:
		nextv = sv.v.IncPatch()
	}

	return &semv{
		v:                &nextv,
		formattedVersion: sv.formattedVersion,
		vPrefix:          sv.vPrefix,
	}
}
