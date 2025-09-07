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
	encoded := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	// mask value to avoid leaking token in logs
	// ref. https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#masking-a-value-in-a-log
	fmt.Printf("::add-mask::%s\n", encoded)

	c.extraheader = fmt.Sprintf(`http.https://%s/.extraheader="Authorization: Basic %s"`,
		host, encoded)
}

func (c *commander) Cmd(prog string, args []string, env map[string]string) (string, string, error) {
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

func (c *commander) Git(args ...string) (string, string, error) {
	return c.Cmd(c.getGitPath(), args, nil)
}
