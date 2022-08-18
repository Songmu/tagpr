package rcpr

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

const versionRegBase = `(?i)((?:^|[^-_0-9a-zA-Z])version[^-_0-9a-zA-Z].{0,20})`

var (
	versionReg = regexp.MustCompile(versionRegBase + `([0-9]+\.[0-9]+\.[0-9]+)`)
	// The "testdata" directory is ommited because of the test code for rcpr itself
	skipDirs = map[string]bool{
		".git":         true,
		"testdata":     true,
		"node_modules": true,
		"vendor":       true,
		"third_party":  true,
		"extlib":       true,
	}
	skipFiles = map[string]bool{
		"requirements.txt":  true,
		"cpanfile.snapshot": true,
		"package-lock.json": true,
	}
)

func isSkipFile(n string) bool {
	n = strings.ToLower(n)
	return strings.HasSuffix(n, ".lock") || skipFiles[n]
}

func detectVersionFile(root string, ver *semv) (string, error) {
	verReg, err := regexp.Compile(versionRegBase + regexp.QuoteMeta(ver.Naked()))
	if err != nil {
		return "", err
	}

	errorCb := func(fpath string, err error) error {
		// When running a rcpr binary under the repository root, "text file busy" occurred,
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
		joinedPath := filepath.Join(root, fpath)
		bs, err := os.ReadFile(joinedPath)
		if err != nil {
			return errorCb(fpath, err)
		}
		if verReg.Match(bs) {
			fl.append(filepath.ToSlash(joinedPath))
		}
		return nil
	}, walker.WithErrorCallback(errorCb)); err != nil {
		return "", err
	}
	list := fl.list()
	if len(list) < 1 {
		return "", nil
	}
	list = fileOrder(list)

	return list[0], nil
	// XXX: Currently, version file detection methods are inaccurate; it might be better to limit it to
	// gemspec, setup.py, setup.cfg, package.json, META.json, and so on. However, there may be cases
	// where some projects have their own version files, and it is annoying to deal with various
	// languages, etc. one by one, so this is the way to go. We would improve it.
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
	verReg, err := regexp.Compile(versionRegBase + regexp.QuoteMeta(from.Naked()))
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
	m := versionReg.FindSubmatch(bs)
	if len(m) < 3 {
		return nil, fmt.Errorf("no version detected from file: %s", fpath)
	}
	ver := string(m[2])
	if vPrefix {
		ver = "v" + ver
	}
	return newSemver(ver)
}
