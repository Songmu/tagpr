package tagpr

import (
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v93/github"
)

func tagLogArgs(prefix string, args ...string) []string {
	if prefix != "" {
		args = append(args, "--", strings.TrimSuffix(prefix, "/"))
	}
	return args
}

func mergedFeatureHeads(mergeLog string) []string {
	var heads []string
	for line := range strings.SplitSeq(mergeLog, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			heads = append(heads, fields[1])
		}
	}
	return heads
}

func mergedPullRequestNumbers(mergedHeads []string, pullRefs string) []int {
	var numbers []int
	for line := range strings.SplitSeq(pullRefs, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		for _, head := range mergedHeads {
			if !strings.HasPrefix(fields[0], head) {
				continue
			}
			number, err := strconv.Atoi(strings.Trim(fields[1], "head/rfspul"))
			if err == nil {
				numbers = append(numbers, number)
			}
		}
	}
	return numbers
}

func treeEntriesFromDiff(diff string) ([]*github.TreeEntry, error) {
	var entries []*github.TreeEntry
	for line := range strings.SplitSeq(diff, "\n") {
		if !strings.HasPrefix(line, ":") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}
		mode, status, path := parts[1], parts[4], parts[5]
		switch status {
		case "A", "M":
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			entries = append(entries, &github.TreeEntry{
				Path:    new(path),
				Type:    new("blob"),
				Content: new(string(content)),
				Mode:    new(mode),
			})
		case "D":
			entries = append(entries, &github.TreeEntry{
				SHA:  nil,
				Path: new(path),
				Type: new("blob"),
				Mode: new("100644"),
			})
		}
	}
	return entries, nil
}

func splitVersionFiles(value string) []string {
	files := strings.Split(value, ",")
	for i, file := range files {
		files[i] = strings.TrimSpace(file)
	}
	return files
}
