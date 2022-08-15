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

func git(args ...string) (string, string, error) {
	log.Println(args)
	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)
	cmd := exec.Command("git", args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()

	if err != nil {
		log.Println(err)
		log.Println(outBuf.String())
		log.Println(errBuf.String())
	}
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}

type cmd struct {
	outStream, errStream io.Writer
	dir                  string
	err                  error
}

func (c *cmd) git(args ...string) (string, string) {
	log.Println(args)
	return c.run("git", args...)
}

func (c *cmd) run(prog string, args ...string) (string, string) {
	if c.err != nil {
		return "", ""
	}
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
	c.err = cmd.Run()
	if c.err != nil {
		log.Println(c.err)
		log.Println(outBuf.String())
		log.Println(errBuf.String())
	}
	return outBuf.String(), errBuf.String()
}

var headBranchReg = regexp.MustCompile(`(?m)^\s*HEAD branch: (.*)$`)

func defaultBranch(remote string) (string, error) {
	if remote == "" {
		var err error
		remote, err = detectRemote()
		if err != nil {
			return "", err
		}
	}
	// `git symbolic-ref refs/remotes/origin/HEAD` sometimes doesn't work
	// So use `git remote show origin` for detecting default branch
	show, _, err := git("remote", "show", remote)
	if err != nil {
		return "", fmt.Errorf("failed to detect defaut branch: %w", err)
	}
	m := headBranchReg.FindStringSubmatch(show)
	if len(m) < 2 {
		return "", fmt.Errorf("failed to detect default branch from remote: %s", remote)
	}
	return m[1], nil
}

func detectRemote() (string, error) {
	remotesStr, _, err := git("remote")
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
