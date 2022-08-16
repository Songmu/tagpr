package rcpr

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

type commander struct {
	outStream, errStream io.Writer
	dir                  string

	err error
}

func (c *commander) cmdE(prog string, args ...string) (string, string, error) {
	if c.err != nil {
		return "", "", c.err
	}
	log.Println(prog, args)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)
	cmd := exec.Command(prog, args...)
	cmd.Stdout = io.MultiWriter(&outBuf, c.outStream)
	cmd.Stderr = io.MultiWriter(&errBuf, c.errStream)
	if c.dir != "" {
		cmd.Dir = c.dir
	}
	err := cmd.Run()
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}

func (c *commander) gitE(args ...string) (string, string, error) {
	return c.cmdE("git", args...)
}

func (c *commander) git(args ...string) (string, string) {
	return c.cmd("git", args...)
}

func (c *commander) cmd(prog string, args ...string) (string, string) {
	stdout, stderr, err := c.cmdE(prog, args...)
	c.err = err
	return stdout, stderr
}

var headBranchReg = regexp.MustCompile(`(?m)^\s*HEAD branch: (.*)$`)

func (rp *rcpr) defaultBranch(remote string) (string, error) {
	if remote == "" {
		var err error
		remote, err = rp.detectRemote()
		if err != nil {
			return "", err
		}
	}
	// `git symbolic-ref refs/remotes/origin/HEAD` sometimes doesn't work
	// So use `git remote show origin` for detecting default branch
	show, _, err := rp.c.gitE("remote", "show", remote)
	if err != nil {
		return "", fmt.Errorf("failed to detect defaut branch: %w", err)
	}
	m := headBranchReg.FindStringSubmatch(show)
	if len(m) < 2 {
		return "", fmt.Errorf("failed to detect default branch from remote: %s", remote)
	}
	return m[1], nil
}

func (rp *rcpr) detectRemote() (string, error) {
	remotesStr, _, err := rp.c.gitE("remote")
	if err != nil {
		return "", fmt.Errorf("failed to detect remote: %s", err)
	}
	remotes := strings.Fields(remotesStr)
	if len(remotes) == 1 {
		return remotes[0], nil
	}
	for _, r := range remotes {
		if r == "origin" {
			return r, nil
		}
	}
	return "", errors.New("failed to detect remote")
}
