package tagpr

import (
	"testing"
)

func TestReplaceCompareLink(t *testing.T) {
	for _, tt := range []struct {
		name, host, current, next, branch, before, after string
	}{
		{
			name: "first prefixed calendar release with unrelated previous tag", host: "github.com",
			next: "docs/2026.0823.0", branch: "tagpr-from-docs-0.0.0",
			before: "compare/v1.20.0...docs/2026.0823.0", after: "commits/tagpr-from-docs-0.0.0",
		},
		{
			name: "first prefixed semantic release", host: "github.com",
			next: "sub/v0.1.0", branch: "tagpr-from-sub-v0.0.0",
			before: "compare/other/v2.0.0...sub/v0.1.0", after: "commits/tagpr-from-sub-v0.0.0",
		},
		{
			name: "first repository release", host: "github.com",
			next: "v0.1.0", branch: "tagpr-from-v0.0.0",
			before: "commits/v0.1.0", after: "commits/tagpr-from-v0.0.0",
		},
		{
			name: "later prefixed release", host: "github.com", current: "docs/2026.0823.0",
			next: "docs/2026.0824.0", branch: "tagpr-from-docs-2026.0823.0",
			before: "compare/docs/2026.0823.0...docs/2026.0824.0", after: "compare/docs/2026.0823.0...tagpr-from-docs-2026.0823.0",
		},
		{
			name: "enterprise host and literal regexp characters", host: "github.example.com",
			next: "sub/v1.0.0+build.1", branch: "tagpr-from-sub-$candidate",
			before: "compare/v2.0.0...sub/v1.0.0+build.1", after: "commits/tagpr-from-sub-$candidate",
		},
		{
			name: "different target stays unchanged", host: "github.com",
			next: "docs/2026.0823.0", branch: "candidate",
			before: "compare/v1.20.0...docs/2026.0824.0", after: "compare/v1.20.0...docs/2026.0824.0",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			base := "**Full Changelog**: https://" + tt.host + "/owner/repo/"
			orig := "## What's Changed\n* Keep this note\n\n" + base + tt.before + "\n"
			want := "## What's Changed\n* Keep this note\n\n" + base + tt.after + "\n"
			got := replaceCompareLink(orig, tt.host, "owner", "repo", tt.current, tt.next, tt.branch)
			if got != want {
				t.Errorf("replaceCompareLink() = %q, want %q", got, want)
			}
		})
	}
}

func TestReplaceCompareLinkPreservesOtherLinks(t *testing.T) {
	orig := "**Full Changelog**: https://github.com/other/repo/compare/v1.0.0...docs/v0.1.0\n" +
		"Example: https://github.com/owner/repo/compare/v1.0.0...docs/v0.1.0\n"
	if got := replaceCompareLink(orig, "github.com", "owner", "repo", "", "docs/v0.1.0", "candidate"); got != orig {
		t.Errorf("unrelated links changed: %q", got)
	}
}
