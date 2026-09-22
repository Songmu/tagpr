package tagpr

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
)

const cmdName = "tagpr"

func printVersion(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%s v%s (rev:%s)\n", cmdName, version, revision)
	return err
}

// Run the tagpr
func Run(ctx context.Context, argv []string, outStream, errStream io.Writer) error {
	log.SetOutput(errStream)
	fs := flag.NewFlagSet(
		fmt.Sprintf("%s (v%s rev:%s)", cmdName, version, revision), flag.ContinueOnError)
	fs.SetOutput(errStream)
	ver := fs.Bool("version", false, "display version")
	mode := fs.String("mode", string(executionModeAuto), "execution mode: auto, prepare, or tag")
	pendingTag := fs.String("pending-tag", "", "expected tag to create in tag mode")
	targetSHA := fs.String("target-sha", "", "commit SHA to tag in tag mode")
	releaseBoundarySHA := fs.String(
		"release-boundary-sha", "", "release notes boundary SHA in tag mode")
	pullRequestNumber := fs.Int(
		"pull-request-number", 0, "merged release pull request number in tag mode")
	baseTag := fs.String("base-tag", "", "previous release tag in tag mode")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if *ver {
		return printVersion(outStream)
	}
	opts := runOptions{
		Mode: executionMode(*mode),
		Candidate: releaseCandidate{
			PendingTag:         *pendingTag,
			TargetSHA:          *targetSHA,
			ReleaseBoundarySHA: *releaseBoundarySHA,
			PullRequestNumber:  *pullRequestNumber,
			BaseTag:            *baseTag,
		},
	}
	if err := opts.validate(); err != nil {
		return err
	}

	tp, err := newTagPR(ctx, &commander{
		gitPath: "git", outStream: outStream, errStream: errStream, dir: "."})
	if err != nil {
		return err
	}
	tp.runOptions = opts
	return tp.Run(ctx)
}
