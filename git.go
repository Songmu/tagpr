package tagpr

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type commander struct {
	outStream, errStream io.Writer
	gitPath, dir         string

	extraheader string
}

func (c *commander) getGitPath() string {
	if c.gitPath == "" {
		return "git"
	}
	return c.gitPath
}

func (c *commander) SetToken(token, host string) {
	confkey := fmt.Sprintf(`http.https://%s/.extraheader`, host)
	if v, _, _ := c.Git("config", confkey); v != "" {
		// XXX: Ideally, we should verify whether the AUTHORIZATION header is present, but for now, we'll proceed with this approach.
		return
	}
	encoded := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	// mask value to avoid leaking token in logs
	// ref. https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#masking-a-value-in-a-log
	fmt.Printf("::add-mask::%s\n", encoded)

	c.extraheader = fmt.Sprintf(`%s=AUTHORIZATION: basic %s`, confkey, encoded)
}

func (c *commander) Cmd(prog string, args []string, env map[string]string) (
	string, string, error) {

	log.Println(prog, args)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)
	cmd := exec.Command(prog, args...)
	if env != nil {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	cmd.Stdout = io.MultiWriter(&outBuf, c.outStream)
	cmd.Stderr = io.MultiWriter(&errBuf, c.errStream)
	if c.dir != "" {
		cmd.Dir = c.dir
	}
	err := cmd.Run()
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}

var needsExtra = map[string]bool{
	"clone":     true,
	"fetch":     true,
	"pull":      true,
	"push":      true,
	"ls-remote": true,
	"submodule": true,
}

func (c *commander) Git(args ...string) (string, string, error) {
	if len(args) > 0 && needsExtra[args[0]] && c.extraheader != "" {
		args = append([]string{"-c", c.extraheader}, args...)
	}
	return c.Cmd(c.getGitPath(), args, nil)
}
