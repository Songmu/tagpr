package gh2changelog

import "testing"

func Mock(t *testing.T, vers []string, g gitter, gen releaseNoteGenerator) Option {
	t.Helper()
	return func(gch *GH2Changelog) {
		gch.semvers = vers
		gch.c = g
		gch.gen = gen
	}
}
