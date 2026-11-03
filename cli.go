package tagpr

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
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
	mode := fs.String("mode", string(executionModeAuto), "execution mode: auto, plan, or apply")
	planJSON := fs.String("plan", "", "release plan JSON emitted by plan mode")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if *ver {
		return printVersion(outStream)
	}
	opts := runOptions{Mode: executionMode(*mode)}
	if *planJSON != "" {
		if err := json.Unmarshal([]byte(*planJSON), &opts.Plan); err != nil {
			return fmt.Errorf("failed to parse --plan: %w", err)
		}
		if opts.Plan == nil {
			return fmt.Errorf("--plan must be a JSON object")
		}
	}
	if err := opts.validate(); err != nil {
		return err
	}

	commandOutput := outStream
	if _, ok := os.LookupEnv("GITHUB_OUTPUT"); !ok && opts.Mode == executionModePlan {
		commandOutput = errStream
	}
	tp, err := newTagPR(ctx, &commander{
		gitPath: "git", outStream: commandOutput, errStream: errStream, dir: "."})
	if err != nil {
		return err
	}
	tp.out = outStream
	tp.runOptions = opts
	return tp.Run(ctx)
}
