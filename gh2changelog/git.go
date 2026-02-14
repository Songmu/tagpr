package gh2changelog

import (
	"bytes"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
)

type gitter interface {
	Git(...string) (string, string, error)
}

var _ gitter = (*commander)(nil)

type commander struct {
	outStream, errStream io.Writer
	gitPath, dir         string

	l          *log.Logger
	initLogger sync.Once
}

func (c *commander) getGitPath() string {
	if c.gitPath == "" {
		return "git"
	}
	return c.gitPath
}

func (c *commander) logger() *log.Logger {
	c.initLogger.Do(func() {
		c.l = log.New(c.errStream, "", 0)
	})
	return c.l
}

func (c *commander) Git(args ...string) (string, string, error) {
	prog := c.getGitPath()
	c.logger().Println(prog, args)
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
