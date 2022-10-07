package tagpr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/saracen/walker"
)

const (
	versionRegBase = `(?i)((?:^|[^-_0-9a-zA-Z])version[^-_0-9a-zA-Z].{0,50}?)`
	semverRegBase  = `([0-9]+\.[0-9]+\.[0-9]+)`
)

var (
	versionReg         = regexp.MustCompile(versionRegBase + semverRegBase)
	versionRegFallback = regexp.MustCompile(semverRegBase)
	skipDirs           = map[string]bool{
		// The "testdata" directory is ommited because of the test code for tagpr itself
		"testdata":     true,
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"third_party":  true,
		"extlib":       true,
		"docs":         true,
		// The directory for storing python test code, but it may be inappropriate to omit this directory
		// uniformly because it may be used by test libraries for other languages.
		"tests": true,
	}
	skipFiles = map[string]bool{
		"requirements.txt":  true,
		"cpanfile.snapshot": true,
		"package-lock.json": true,
	}
	skipExt = map[string]bool{
		".md":   true,
		".rst":  true,
		".adoc": true,
	}
)

func isSkipFile(n string) bool {
	n = strings.ToLower(n)
	return strings.HasSuffix(n, ".lock") || skipFiles[n] || skipExt[filepath.Ext(n)]
}

func detectVersionFile(root string, ver *semv) (string, error) {
	verReg, err := regexp.Compile(versionRegBase + regexp.QuoteMeta(ver.Naked()))
	if err != nil {
		return "", err
	}

	errorCb := func(fpath string, err error) error {
		// When running a tagpr binary under the repository root, "text file busy" occurred,
		// so I did error handling as this, but it did not solve the problem, and it is a special case,
		// so we may not need to do the check in particular.
		if os.IsPermission(err) || errors.Is(err, syscall.ETXTBSY) {
			return nil
		}
		return err
	}

	fl := &fileList{}
	if err := walker.Walk(root, func(fpath string, fi os.FileInfo) error {
		if fi.IsDir() {
			if skipDirs[fi.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !fi.Mode().IsRegular() || isSkipFile(fi.Name()) {
			return nil
		}
		bs, err := os.ReadFile(fpath)
		if err != nil {
			return errorCb(fpath, err)
		}
		if verReg.Match(bs) {
			f, _ := filepath.Rel(root, fpath)
			fl.append(filepath.ToSlash(f))
		}
		return nil
	}, walker.WithErrorCallback(errorCb)); err != nil {
		return "", err
	}

	// XXX: Whether to adopt a version file when the language is not identifiable?
	f, _ := versionFile(fl.list())
	return f, nil
}

// The second argument returns the language name in lowercase, but this is a temporary interface
// and is not currently used anywhere.
func versionFile(files []string) (string, string) {
	if len(files) < 1 {
		return "", ""
	}
	files = fileOrder(files)
	var meta string
	for _, f := range files {
		if strings.HasSuffix(f, ".gemspec") {
			return f, "ruby"
		}
		if strings.HasSuffix(f, ".go") {
			return f, "go"
		}
		if meta != "" {
			if strings.HasPrefix(f, "lib/") && strings.HasSuffix(f, ".pm") {
				return f, "perl"
			}
		}

		base := strings.ToLower(filepath.Base(f))
		switch base {
		case "setup.py", "setup.cfg":
			return f, "python"
		case "package.json":
			return f, "node"
		case "manifest.json": // for chrome extension
			return f, "javascript"
		case "pom.xml":
			return f, "java"
		case "meta.json":
			if meta == "" {
				meta = f
			}
		}
	}

	if meta != "" {
		return meta, "perl"
	}
	return files[0], ""
}

func fileOrder(list []string) []string {
	sort.Slice(list, func(i, j int) bool {
		x := list[i]
		y := list[j]
		xdepth := strings.Count(x, "/")
		ydepth := strings.Count(y, "/")
		if xdepth != ydepth {
			return xdepth < ydepth
		}
		return strings.Compare(x, y) < 0
	})
	return list
}

type fileList struct {
	l  []string
	mu sync.RWMutex
}

func (fl *fileList) append(fpath string) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.l = append(fl.l, fpath)
}

func (fl *fileList) list() []string {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	return fl.l
}

func bumpVersionFile(fpath string, from, to *semv) error {
	verReg, err := regexp.Compile(`(v|\b)` + regexp.QuoteMeta(from.Naked()) + `\b`)
	if err != nil {
		return err
	}
	bs, err := os.ReadFile(fpath)
	if err != nil {
		return err
	}

	replaced := false
	updated := verReg.ReplaceAllFunc(bs, func(match []byte) []byte {
		if replaced {
			return match
		}
		replaced = true
		return verReg.ReplaceAll(match, []byte(`${1}`+to.Naked()))
	})
	return os.WriteFile(fpath, updated, 0666)
}

func retrieveVersionFromFile(fpath string, vPrefix bool) (*semv, error) {
	bs, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	var ver string
	if m := versionReg.FindSubmatch(bs); len(m) >= 3 {
		ver = string(m[2])
	} else {
		m := versionRegFallback.FindSubmatch(bs)
		if len(m) < 2 {
			return nil, fmt.Errorf("no version detected from file: %s", fpath)
		}
		ver = string(m[1])
	}
	if vPrefix {
		ver = "v" + ver
	}
	return newSemver(ver)
}
