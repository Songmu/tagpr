package tagpr

import (
	"time"

	"github.com/Masterminds/semver/v3"
)

type semv struct {
	v *semver.Version

	vPrefix           bool
	asCalendarVersion bool
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
	if sv.asCalendarVersion {
		return sv.nextCalver(time.Now())
	}

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

func newCalver(now time.Time, vPrefix bool) *semv {
	major := uint64(now.Year())                          // YYYY
	minor := uint64(now.Month()*100) + uint64(now.Day()) // MMDD without leading zeros
	v := semver.New(major, minor, uint64(0), "", "")
	return &semv{
		v:                 v,
		vPrefix:           vPrefix,
		asCalendarVersion: true,
	}
}

func (sv *semv) nextCalver(now time.Time) *semv {
	curr := newCalver(now, sv.vPrefix)
	if sv.v.Major() != curr.v.Major() || sv.v.Minor() != curr.v.Minor() {
		// Another date. Reset patch to 0
		return curr
	}
	// Same date. Increment patch
	nextv := sv.v.IncPatch()
	return &semv{
		v:                 &nextv,
		vPrefix:           sv.vPrefix,
		asCalendarVersion: true,
	}
}
