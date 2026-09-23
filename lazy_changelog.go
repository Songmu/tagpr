package tagpr

import (
	"sync"

	"github.com/Songmu/tagpr/gh2changelog"
)

type draftChangelog struct {
	generator      *gh2changelog.GH2Changelog
	changelog      string
	generatedNotes string
}

type lazyDraftChangelog struct {
	once   sync.Once
	load   func() (*draftChangelog, error)
	result *draftChangelog
	err    error
}

func (l *lazyDraftChangelog) Load() (*draftChangelog, error) {
	l.once.Do(func() {
		l.result, l.err = l.load()
	})
	return l.result, l.err
}

func (l *lazyDraftChangelog) GeneratedNotes() (string, error) {
	result, err := l.Load()
	if err != nil {
		return "", err
	}
	return result.generatedNotes, nil
}
