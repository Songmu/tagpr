package rcpr

import (
	"bytes"
	"io"
	"log"
	"os/exec"
	"strings"
)

type commander struct {
	outStream, errStream io.Writer
	gitPath, dir         string

	err error
}

func (c *commander) getGitPath() string {
	if c.gitPath == "" {
		return "git"
	}
	return c.gitPath
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

func (c *commander) GitE(args ...string) (string, string, error) {
	return c.cmdE(c.getGitPath(), args...)
}

func (c *commander) Git(args ...string) (string, string) {
	return c.cmd(c.getGitPath(), args...)
}

func (c *commander) cmd(prog string, args ...string) (string, string) {
	stdout, stderr, err := c.cmdE(prog, args...)
	c.err = err
	return stdout, stderr
}
