package rcpr

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
