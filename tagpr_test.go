package tagpr

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-github/v47/github"
)

func TestBuildChunkSearchIssuesQuery(t *testing.T) {
	tests := []struct {
		queryBase string
		shasStr   string
		want      []string
	}{
		{
			"repo:/Songmu/tagpr is:pr is:closed",
			"",
			nil,
		},
		{
			"repo:/Songmu/tagpr is:pr is:closed",
			`
			`,
			nil,
		},
		{
			"repo:/Songmu/tagpr is:pr is:closed",
			`aeed69aa554533dcb4332a1778e4771165a909b5
`,
			[]string{"repo:/Songmu/tagpr is:pr is:closed aeed69aa554533dcb4332a1778e4771165a909b5"},
		},
		{
			"repo:Songmu/tagpr is:pr is:closed",
			`1a8bb97
1b7691b
a9462b9
4d2b5e9
9ce4268
1eccbf8
1c3fbfc
968ade5
531c782
780bb71
6025fbf
cc369ba
a1f3e39
792bc85
3e3c4e1
37832de
ac97702
d742186
217eb5d
0f900f7
5ef33d1
1d2ec15
2f37752
066ad7b
2e19b14
52b3706
f5134ae
ea39bbf
76b0630
ee3c6e6
2336be4
423a209
63caa74
3296052
3c98d78
86b8739
2264ec5
5c1d87b
4ffe09c
7c5d0de
3de9ed0
1b6b58c
2b643ec
53bf089
e8e96d5
3dac4b0
0605ba4
86cb76d
358c7c1
a139f86
33c16b6
c91f8ff
a109671
b4029bd
f985b4f
b74ef35
53d9ab3
6f57b07
0a84d90
43aa57d
75b6f79
def3db8
c0fc143
`,
			[]string{
				"repo:Songmu/tagpr is:pr is:closed 1a8bb97 1b7691b a9462b9 4d2b5e9 9ce4268 1eccbf8 1c3fbfc 968ade5 531c782 780bb71 6025fbf cc369ba a1f3e39 792bc85 3e3c4e1 37832de ac97702 d742186 217eb5d 0f900f7 5ef33d1 1d2ec15 2f37752 066ad7b 2e19b14 52b3706 f5134ae",
				"repo:Songmu/tagpr is:pr is:closed ea39bbf 76b0630 ee3c6e6 2336be4 423a209 63caa74 3296052 3c98d78 86b8739 2264ec5 5c1d87b 4ffe09c 7c5d0de 3de9ed0 1b6b58c 2b643ec 53bf089 e8e96d5 3dac4b0 0605ba4 86cb76d 358c7c1 a139f86 33c16b6 c91f8ff a109671 b4029bd",
				"repo:Songmu/tagpr is:pr is:closed f985b4f b74ef35 53d9ab3 6f57b07 0a84d90 43aa57d 75b6f79 def3db8 c0fc143",
			},
		},
		{
			"repo:Songmu/tagpr is:pr is:closed",
			`1a8bb97
1b7691b
a9462b9
4d2b5e9
9ce4268
1eccbf8
1c3fbfc
968ade5
531c782
780bb71
6025fbf
cc369ba
a1f3e39
792bc85
3e3c4e1
37832de
ac97702
d742186
217eb5d
0f900f7
5ef33d1
1d2ec15
2f37752
066ad7b
2e19b14
52b3706
f5134ae
`,
			[]string{
				"repo:Songmu/tagpr is:pr is:closed 1a8bb97 1b7691b a9462b9 4d2b5e9 9ce4268 1eccbf8 1c3fbfc 968ade5 531c782 780bb71 6025fbf cc369ba a1f3e39 792bc85 3e3c4e1 37832de ac97702 d742186 217eb5d 0f900f7 5ef33d1 1d2ec15 2f37752 066ad7b 2e19b14 52b3706 f5134ae",
			},
		},
	}
	for _, tt := range tests {
		got := buildChunkSearchIssuesQuery(tt.queryBase, tt.shasStr)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("got:\n%s,\nwant:\n%s", got, tt.want)
		}
	}
}

func TestGeneratenNextLabels(t *testing.T) {
	major := "major"
	minor := "minor"
	enhancement := "enhancement"
	breakingChange := "breaking-change"

	tests := map[string]struct {
		labels      []*github.Label
		majorLabels string
		minorLabels string
		want        []string
	}{
		"onlyMajor": {
			[]*github.Label{newGithubLabel(&major)},
			"",
			"",
			[]string{"tagpr:major"},
		},
		"onlyMinor": {
			[]*github.Label{newGithubLabel(&minor)},
			"",
			"",
			[]string{"tagpr:minor"},
		},
		"other": {
			[]*github.Label{newGithubLabel(new(string))},
			"",
			"",
			[]string{},
		},
		"enhancement": {
			[]*github.Label{newGithubLabel(&enhancement)},
			"",
			"",
			[]string{},
		},
		"breakingChange": {
			[]*github.Label{newGithubLabel(&breakingChange)},
			"",
			"",
			[]string{},
		},
		"empty": {
			[]*github.Label{},
			"",
			"",
			[]string{},
		},
		"majorAndMinor": {
			[]*github.Label{newGithubLabel(&major), newGithubLabel(&minor)},
			"",
			"",
			[]string{"tagpr:minor", "tagpr:major"},
		},
		"minorAndMajor": {
			[]*github.Label{newGithubLabel(&minor), newGithubLabel(&major)},
			"",
			"",
			[]string{"tagpr:minor", "tagpr:major"},
		},
		"Set breakingChange to majorLabels": {
			[]*github.Label{newGithubLabel(&breakingChange)},
			breakingChange,
			"",
			[]string{"tagpr:major"},
		},
		"Set enhancement to minorLabels": {
			[]*github.Label{newGithubLabel(&enhancement)},
			"",
			enhancement,
			[]string{"tagpr:minor"},
		},
		"Include in majorLabels to breakingChange": {
			[]*github.Label{newGithubLabel(&breakingChange)},
			fmt.Sprintf("other, %s", breakingChange),
			"",
			[]string{"tagpr:major"},
		},
		"Include in minorLabels to enhancement": {
			[]*github.Label{newGithubLabel(&enhancement)},
			"",
			fmt.Sprintf(" %s,other", enhancement),
			[]string{"tagpr:minor"},
		},
		"Invalid default label": {
			[]*github.Label{newGithubLabel(&major), newGithubLabel(&minor)},
			breakingChange,
			enhancement,
			[]string{},
		},
		"Include default label": {
			[]*github.Label{newGithubLabel(&major), newGithubLabel(&enhancement)},
			"major,breaking-change",
			"minor,enhancement",
			[]string{"tagpr:minor", "tagpr:major"},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			if tt.majorLabels != "" {
				t.Setenv(envMajorLabels, tt.majorLabels)
			}
			if tt.minorLabels != "" {
				t.Setenv(envMinorLabels, tt.minorLabels)
			}
			tp, err := newTagPR(context.Background(), &commander{
				gitPath: "git", outStream: os.Stdout, errStream: os.Stderr, dir: "."},
			)
			if err != nil {
				t.Error(err)
			}

			prIssues := []*github.Issue{
				{
					ID:                new(int64),
					Number:            new(int),
					State:             new(string),
					Locked:            new(bool),
					Title:             new(string),
					Body:              new(string),
					AuthorAssociation: new(string),
					User:              &github.User{},
					Labels:            tt.labels,
					Assignee:          &github.User{},
					Comments:          new(int),
					ClosedAt:          &time.Time{},
					CreatedAt:         &time.Time{},
					UpdatedAt:         &time.Time{},
					ClosedBy:          &github.User{},
					URL:               new(string),
					HTMLURL:           new(string),
					CommentsURL:       new(string),
					EventsURL:         new(string),
					LabelsURL:         new(string),
					RepositoryURL:     new(string),
					Milestone:         &github.Milestone{},
					PullRequestLinks:  &github.PullRequestLinks{},
					Repository:        &github.Repository{},
					Reactions:         &github.Reactions{},
					Assignees:         []*github.User{},
					NodeID:            new(string),
					TextMatches:       []*github.TextMatch{},
					ActiveLockReason:  new(string),
				},
			}

			got := tp.generatenNextLabels(prIssues)

			if len(got) == 0 && len(tt.want) == 0 {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:\n%s,\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func newGithubLabel(name *string) *github.Label {
	return &github.Label{
		ID:          new(int64),
		URL:         new(string),
		Name:        name,
		Color:       new(string),
		Description: new(string),
		Default:     new(bool),
		NodeID:      new(string),
	}
}
