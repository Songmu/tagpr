package tagpr

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v57/github"
)

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func (tp *tagpr) setOutput(name, value string) error {
	return setOutput(name, value)
}

func setOutput(name, value string) error {
	fpath, ok := os.LookupEnv("GITHUB_OUTPUT")
	if !ok {
		return nil
	}
	f, err := os.OpenFile(fpath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s=%s\n", name, value))
	return err
}

func showGHError(err error, resp *github.Response) {
	title := "failed to request GitHub API"
	message := err.Error()
	if resp != nil {
		respInfo := []string{
			fmt.Sprintf("status=%d", resp.StatusCode),
		}
		for name, values := range resp.Header {
			if strings.HasPrefix(strings.ToLower(name), "x-ratelimit") {
				respInfo = append(respInfo, fmt.Sprintf("%s=%v", name, values))
			}
		}
		message += " " + strings.Join(respInfo, ",")
	}
	// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#setting-an-error-message
	fmt.Printf("::error title=%s::%s\n", title, message)
}
