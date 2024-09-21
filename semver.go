package tagpr

import "github.com/Masterminds/semver/v3"

type semv struct {
	v *semver.Version

	vPrefix bool
}

func newSemver(v string) (*semv, error) {
	var err error
	sv := &semv{}
	sv.v, err = semver.NewVersion(v)
	if err != nil {
		return nil, err
	}
	sv.vPrefix = v[0] == 'v'
	return sv, nil
}

func (sv *semv) Naked() string {
	return sv.v.String()
}

func (sv *semv) Tag() string {
	if sv.vPrefix {
		return "v" + sv.Naked()
	}
	return sv.Naked()
}

func (sv *semv) GuessNext(labels []string) *semv {
	var isMajor, isMinor bool
	for _, l := range labels {
		switch l {
		case autoLabelName + ":major", autoLabelName + "/major":
			isMajor = true
		case autoLabelName + ":minor", autoLabelName + "/minor":
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
		v:       &nextv,
		vPrefix: sv.vPrefix,
	}
}
