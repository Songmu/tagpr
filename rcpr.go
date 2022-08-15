package rcpr

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/Songmu/gitsemvers"
	"github.com/google/go-github/v45/github"
)

const cmdName = "rcpr"

var remoteReg = regexp.MustCompile(`origin\s.*?github\.com[:/]([-a-zA-Z0-9]+)/(\S+)`)

type rcpr struct {
}

func (rp *rcpr) latestSemverTag() string {
	vers := (&gitsemvers.Semvers{}).VersionStrings()
	if len(vers) > 0 {
		return vers[0]
	}
	return ""
}

// Run the rcpr
func Run(ctx context.Context, argv []string, outStream, errStream io.Writer) error {
	log.SetOutput(errStream)
	fs := flag.NewFlagSet(
		fmt.Sprintf("%s (v%s rev:%s)", cmdName, version, revision), flag.ContinueOnError)
	fs.SetOutput(errStream)
	ver := fs.Bool("version", false, "display version")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if *ver {
		return printVersion(outStream)
	}

	rp := &rcpr{}
	currVer := rp.latestSemverTag()
	if currVer == "" {
		currVer = "v0.0.0"
	}

	releaseBranch, _ := defaultBranch("") // TODO: make configable
	if releaseBranch == "" {
		releaseBranch = "main"
	}
	branch, _, err := git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to release when git symbolic-ref: %w", err)
	}
	if branch != releaseBranch {
		return fmt.Errorf("you are not on releasing branch %q, current branch is %q",
			releaseBranch, branch)
	}

	rcBranch := fmt.Sprintf("rcpr-%s", currVer)
	git("branch", "-D", rcBranch)

	c := &cmd{outStream: outStream, errStream: errStream, dir: "."}
	c.git("config", "--local", "user.email", "github-actions[bot]@users.noreply.github.com")
	c.git("config", "--local", "user.name", "github-actions[bot]")

	c.git("checkout", "-b", rcBranch)

	// XXX do some releng related changes before commit
	c.git("commit", "--allow-empty", "-am", "release")

	// TODO: If remote rc branches are advanced, apply them with cherry-pick, etc.

	c.git("push", "--force", "origin", rcBranch)
	if c.err != nil {
		return c.err
	}

	remote, _, err := git("remote", "-v")
	if err != nil {
		return err
	}
	m := remoteReg.FindStringSubmatch(remote)
	if len(m) < 3 {
		return fmt.Errorf("failed to detect remote")
	}
	owner := m[1]
	repo := m[2]
	// XXX: This is to remove the ".git" suffix of the git schema or scp like URL,
	// but if the repository name really ends in .git, it will be removed, but it's OK for now.
	repo = strings.TrimSuffix(repo, ".git")

	cli, err := client(ctx, "", "")
	if err != nil {
		return err
	}
	v, err := semver.NewVersion(currVer)
	if err != nil {
		return err
	}
	nextVer := "v" + v.IncPatch().String() // XXX: proper next version detection

	previousTag := &currVer
	if *previousTag == "v0.0.0" {
		previousTag = nil
	}
	releases, _, err := cli.Repositories.GenerateReleaseNotes(
		ctx, owner, repo, &github.GenerateNotesOptions{
			TagName:         nextVer,
			PreviousTagName: previousTag,
			TargetCommitish: &releaseBranch,
		})
	if err != nil {
		return err
	}

	head := fmt.Sprintf("%s:%s", owner, rcBranch)
	pulls, _, err := cli.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head: head,
		Base: releaseBranch,
	})
	if err != nil {
		return err
	}

	pstr := func(str string) *string {
		return &str
	}
	title := fmt.Sprintf("release %s", nextVer)
	if len(pulls) == 0 {
		pr, _, err := cli.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
			Title: pstr(title),
			Body:  pstr(releases.Body),
			Base:  &releaseBranch,
			Head:  pstr(head),
		})
		if err != nil {
			return err
		}
		_, _, err = cli.Issues.AddLabelsToIssue(ctx, owner, repo, *pr.Number, []string{"rcpr"})
		return err
	}
	pr := pulls[0]
	pr.Title = pstr(title)
	pr.Body = pstr(mergeBody(*pr.Body, releases.Body))
	_, _, err = cli.PullRequests.Edit(ctx, owner, repo, *pr.Number, pr)
	return err
}

func mergeBody(now, update string) string {
	return update
}

func printVersion(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%s v%s (rev:%s)\n", cmdName, version, revision)
	return err
}
