package tagpr

import (
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/k1LoW/calver"
)

type semv struct {
	sv *semver.Version // for SemVer
	cv *calver.Calver  // for CalVer

	vPrefix           bool
	asCalendarVersion bool
	calverFormat      string
	originalVersion   string // original version string for calver parsing
}

func newSemver(v string) (*semv, error) {
	var err error
	sv := &semv{}
	sv.sv, err = semver.NewVersion(v)
	if err != nil {
		return nil, err
	}
	sv.vPrefix = v[0] == 'v'
	sv.originalVersion = v
	return sv, nil
}

func (sv *semv) Naked() string {
	if sv.asCalendarVersion && sv.cv != nil {
		return sv.cv.String()
	}
	return sv.sv.String()
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
		nextv = sv.sv.IncMajor()
	case isMinor:
		nextv = sv.sv.IncMinor()
	default:
		nextv = sv.sv.IncPatch()
	}

	return &semv{
		sv:      &nextv,
		vPrefix: sv.vPrefix,
	}
}

func newCalver(now time.Time, vPrefix bool, format string) *semv {
	cv, _ := calver.NewWithTime(format, now)
	return &semv{
		cv:                cv,
		vPrefix:           vPrefix,
		asCalendarVersion: true,
		calverFormat:      format,
	}
}

func (sv *semv) nextCalver(now time.Time) *semv {
	format := sv.calverFormat
	if format == "" {
		format = defaultCalendarVersioningFormat
	}

	// Use original version string for parsing (preserves zero-padding, etc.)
	verStr := sv.originalVersion
	if verStr == "" {
		verStr = sv.sv.String()
	}
	// Strip v prefix for parsing
	if strings.HasPrefix(verStr, "v") {
		verStr = verStr[1:]
	}

	// Parse current version with the format
	currCv, err := calver.Parse(format, verStr)
	if err != nil {
		// If parsing fails, create new calver from current time
		return newCalver(now, sv.vPrefix, format)
	}

	// Use NextWithTime to get the next version based on the current time
	nextCv, err := currCv.NextWithTime(now)
	if err != nil {
		// If NextWithTime fails (e.g., time is in the past), create new calver
		return newCalver(now, sv.vPrefix, format)
	}

	return &semv{
		cv:                nextCv,
		vPrefix:           sv.vPrefix,
		asCalendarVersion: true,
		calverFormat:      format,
	}
}
