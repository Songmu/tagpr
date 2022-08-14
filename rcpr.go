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

	vers := (&gitsemvers.Semvers{}).VersionStrings()
	currVer := "v0.0.0"
	if len(vers) > 0 {
		currVer = vers[0]
	}
	defaultBr, _ := defaultBranch("") // TODO: make configable
	if defaultBr == "" {
		defaultBr = "main"
	}
	branch, _, err := git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("faild to release when git symbolic-ref: %w", err)
	}
	if branch != defaultBr {
		return fmt.Errorf("you are not on releasing branch %q, current branch is %q",
			defaultBr, branch)
	}

	rcBranch := fmt.Sprintf("rc-%s", currVer)
	git("branch", "-D", rcBranch)

	c := &cmd{outStream: outStream, errStream: errStream, dir: "."}
	c.git("config", "--local", "user.email", "github-actions[bot]@users.noreply.github.com")
	c.git("config", "--local", "user.name", "github-actions[bot]")

	c.git("checkout", "-b", rcBranch)

	// XXX do some releng related changes before commit
	c.git("commit", "--allow-empty", "-am", "release")

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
	repo = strings.TrimSuffix(repo, ".git") // XXX

	cli, err := client(ctx, "", "")
	if err != nil {
		return err
	}

	v, err := semver.NewVersion(currVer)
	if err != nil {
		return err
	}
	nextVer := "v" + v.IncPatch().String() // XXX proper next version detection

	previousTag := &currVer
	if *previousTag == "v0.0.0" {
		previousTag = nil
	}
	releases, _, err := cli.Repositories.GenerateReleaseNotes(
		ctx, owner, repo, &github.GenerateNotesOptions{
			TagName:         nextVer,
			PreviousTagName: previousTag,
			TargetCommitish: &defaultBr,
		})
	if err != nil {
		return err
	}

	pulls, _, err := cli.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:%s", owner, rcBranch),
		Base: defaultBr,
	})
	if err != nil {
		return err
	}

	pstr := func(str string) *string {
		return &str
	}
	if len(pulls) == 0 {
		_, _, err := cli.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
			Title: pstr(fmt.Sprintf("release %s", nextVer)),
			Body:  pstr(releases.Body),
			Base:  &defaultBr,
			Head:  pstr(fmt.Sprintf("%s:%s", owner, rcBranch)),
		})
		return err
	}
	_, _, err = cli.PullRequests.Edit(ctx, owner, repo, *pulls[0].Number, &github.PullRequest{
		Title: pstr(fmt.Sprintf("release %s", nextVer)),
		Body:  pstr(releases.Body),
	})
	return err
}

func printVersion(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%s v%s (rev:%s)\n", cmdName, version, revision)
	return err
}
