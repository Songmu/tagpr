package tagpr

import (
	"fmt"
	"os"
)

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func (tp *tagpr) setOutput(name, value string) error {
	_, err := fmt.Fprintf(tp.out, "::set-output name=%s::%s\n", name, value)
	return err
}
