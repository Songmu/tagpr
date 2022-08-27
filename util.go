package rcpr

import "os"

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
