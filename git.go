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
