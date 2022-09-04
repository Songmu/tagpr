package tagpr

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
}

func (c *commander) getGitPath() string {
	if c.gitPath == "" {
		return "git"
	}
	return c.gitPath
}

func (c *commander) Cmd(prog string, args ...string) (string, string, error) {
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

func (c *commander) Git(args ...string) (string, string, error) {
	return c.Cmd(c.getGitPath(), args...)
}
