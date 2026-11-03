package tagpr

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-github/v93/github"
)

func TestMergedFeatureHeads(t *testing.T) {
	log := "\ninvalid\nbase first\nbase second third\n"
	if got, want := mergedFeatureHeads(log), []string{"first", "second"}; !reflect.DeepEqual(got, want) {
		t.Errorf("mergedFeatureHeads() = %v, want %v", got, want)
	}
}

func TestMergedPullRequestNumbers(t *testing.T) {
	refs := "invalid\nabcdef refs/pull/12/head\nabcdef refs/pull/not-a-number/head\n" +
		"unmatched refs/pull/34/head\nabc123 refs/pull/56/head\nabcdef refs/pull/78/head extra\n"
	if got, want := mergedPullRequestNumbers([]string{"abc", "abcdef"}, refs), []int{12, 12, 56}; !reflect.DeepEqual(got, want) {
		t.Errorf("mergedPullRequestNumbers() = %v, want %v", got, want)
	}
	if got := mergedPullRequestNumbers(nil, refs); got != nil {
		t.Errorf("mergedPullRequestNumbers(nil, refs) = %v, want nil", got)
	}
}

func TestTreeEntriesFromDiff(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	added := "added"
	modified := "modified"
	for path, content := range map[string]string{added: "new", modified: "updated"} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	diff := fmt.Sprintf("\ninvalid\n:malformed\n"+
		":000000 100755 0000000 0000000 A\t%s\n"+
		":100644 100644 0000000 0000000 M\t%s\n"+
		":100755 000000 0000000 0000000 D\tdeleted\n"+
		":100644 100644 0000000 0000000 R100\told\tnew\n", added, modified)
	got, err := treeEntriesFromDiff(diff)
	if err != nil {
		t.Fatal(err)
	}
	want := []*github.TreeEntry{
		{Path: new(added), Type: new("blob"), Content: new("new"), Mode: new("100755")},
		{Path: new(modified), Type: new("blob"), Content: new("updated"), Mode: new("100644")},
		{Path: new("deleted"), Type: new("blob"), Mode: new("100644")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("treeEntriesFromDiff() = %#v, want %#v", got, want)
	}
	t.Run("read failure", func(t *testing.T) {
		diff := ":000000 100644 0000000 0000000 A\tmissing"
		if _, err := treeEntriesFromDiff(diff); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("treeEntriesFromDiff() error = %v, want missing file", err)
		}
	})
}

func TestSplitVersionFiles(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  []string
	}{
		{" alpha.go, beta.json, ,gamma ", []string{"alpha.go", "beta.json", "", "gamma"}},
		{"", []string{""}},
	} {
		t.Run(tt.value, func(t *testing.T) {
			if got := splitVersionFiles(tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitVersionFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}
