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

const (
	cmdName              = "rcpr"
	gitUser              = "github-actions[bot]"
	gitEmail             = "github-actions[bot]@users.noreply.github.com"
	defaultReleaseBranch = "main"
	autoCommitMessage    = "[rcpr] prepare for the next release"
)

type rcpr struct {
}

func (rp *rcpr) latestSemverTag() string {
	vers := (&gitsemvers.Semvers{}).VersionStrings()
	if len(vers) > 0 {
		return vers[0]
	}
	return ""
}

var (
	gitlogReg = regexp.MustCompile(`^([a-f0-9]+)\s(.*)`)
	remoteReg = regexp.MustCompile(`origin\s.*?github\.com[:/]([-a-zA-Z0-9]+)/(\S+)`)
)

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
		releaseBranch = defaultReleaseBranch
	}
	branch, _, err := git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to release when git symbolic-ref: %w", err)
	}
	if branch != releaseBranch {
		return fmt.Errorf("you are not on releasing branch %q, current branch is %q",
			releaseBranch, branch)
	}

	isShallow, _, err := git("rev-parse", "--is-shallow-repository")
	if err != nil {
		return err
	}
	if isShallow == "true" {
		if _, _, err := git("fetch", "--unshallow"); err != nil {
			return err
		}
	}

	rcBranch := fmt.Sprintf("rcpr-%s", currVer)
	git("branch", "-D", rcBranch)

	c := &cmd{outStream: outStream, errStream: errStream, dir: "."}
	c.git("config", "--local", "user.email", gitEmail)
	c.git("config", "--local", "user.name", gitUser)

	c.git("checkout", "-b", rcBranch)

	// XXX do some releng related changes before commit
	c.git("commit", "--allow-empty", "-am", autoCommitMessage)

	// cherry-pick if the remote branch is exists and changed
	out, _, err := git("log", "--no-merges", "--pretty=format:%h %an %s", "main..origin/"+rcBranch)
	if err == nil {
		var cherryPicks []string
		for _, line := range strings.Split(out, "\n") {
			m := gitlogReg.FindStringSubmatch(line)
			if len(m) < 3 {
				continue
			}
			commitish := m[1]
			authorAndSubject := strings.TrimSpace(m[2])
			if authorAndSubject != gitUser+" "+autoCommitMessage {
				cherryPicks = append(cherryPicks, commitish)
			}
		}
		if len(cherryPicks) > 0 {
			for i := len(cherryPicks) - 1; i >= 0; i-- {
				commitish := cherryPicks[i]
				_, _, err := git(
					"cherry-pick", "--keep-redundant-commits", "--allow-empty", commitish)

				// conflict / Need error handling in case of non-conflict error?
				if err != nil {
					git("cherry-pick", "--abort")
				}
			}
		}
	}

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
