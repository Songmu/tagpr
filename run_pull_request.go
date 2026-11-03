package tagpr

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/go-github/v93/github"
)

func (tp *tagpr) updateReleasePullRequestBranchRef(ctx context.Context, rcBranch string, ref *github.Reference, newCommit *github.Commit) error {
	rcBranchRef := "refs/heads/" + rcBranch
	_, resp, err := tp.gh.Git.GetRef(ctx, tp.owner, tp.repo, rcBranchRef)
	if err != nil {
		if resp.StatusCode != 404 {
			tp.showGHError(err, resp)
			return err
		}
		createNewRef := github.CreateRef{
			Ref: "refs/heads/" + rcBranch,
			SHA: *ref.Object.SHA,
		}
		_, resp, err = tp.gh.Git.CreateRef(ctx, tp.owner, tp.repo, createNewRef)
		if err != nil {
			tp.showGHError(err, resp)
			return err
		}
	}
	updateRef := github.UpdateRef{
		SHA:   *newCommit.SHA,
		Force: new(true),
	}
	_, resp, err = tp.gh.Git.UpdateRef(ctx, tp.owner, tp.repo, rcBranchRef, updateRef)
	if err != nil {
		tp.showGHError(err, resp)
		return err
	}

	return nil
}

func (tp *tagpr) publishReleasePullRequest(ctx context.Context, currTagPR *github.PullRequest, prText, releaseBranch, head string, addingLabels []string) error {
	stuffs := strings.SplitN(strings.TrimSpace(prText), "\n", 2)
	title := stuffs[0]
	var body string
	if len(stuffs) > 1 {
		body = strings.TrimSpace(stuffs[1])
	}
	if currTagPR == nil {
		return tp.createReleasePullRequest(ctx, title, body, releaseBranch, head, addingLabels)
	}
	return tp.editReleasePullRequest(ctx, currTagPR, title, body, addingLabels)
}

func (tp *tagpr) createReleasePullRequest(ctx context.Context, title, body, releaseBranch, head string, addingLabels []string) error {
	pr, resp, err := tp.gh.PullRequests.Create(ctx, tp.owner, tp.repo, github.CreatePullRequest{
		Title: new(title),
		Body:  new(body),
		Base:  releaseBranch,
		Head:  head,
	})
	if err != nil {
		tp.showGHError(err, resp)
		return err
	}
	addingLabels = append(addingLabels, autoLabelName)
	_, resp, err = tp.gh.Issues.AddLabelsToIssue(
		ctx, tp.owner, tp.repo, *pr.Number, addingLabels)
	if err != nil {
		tp.showGHError(err, resp)
		return err
	}
	tmpPr, resp, err := tp.gh.PullRequests.Get(ctx, tp.owner, tp.repo, *pr.Number)
	if err == nil {
		pr = tmpPr
	} else {
		tp.showGHError(err, resp)
	}
	b, _ := json.Marshal(pr)
	tp.setOutput("pull_request", string(b))
	return nil
}

func (tp *tagpr) editReleasePullRequest(ctx context.Context, currTagPR *github.PullRequest, title, body string, addingLabels []string) error {
	currTagPR.Title = new(title)
	currTagPR.Body = new(mergeBody(*currTagPR.Body, body))
	// Clear Base so go-github does not include the `base` field in the PATCH
	// payload. When `base` is sent shortly after an UpdateRef (force-push),
	// GitHub emits a duplicate `pull_request.synchronize` webhook, causing
	// pull_request-triggered workflows to run twice on the same SHA.
	// Release PRs never change their base branch, so this is safe.
	// See: https://github.com/google/go-github/issues/1250
	currTagPR.Base = nil
	pr, resp, err := tp.gh.PullRequests.Edit(ctx, tp.owner, tp.repo, *currTagPR.Number, currTagPR)
	if err != nil {
		tp.showGHError(err, resp)
		return err
	}
	if len(addingLabels) > 0 {
		_, resp, err := tp.gh.Issues.AddLabelsToIssue(
			ctx, tp.owner, tp.repo, *currTagPR.Number, addingLabels)
		if err != nil {
			tp.showGHError(err, resp)
			return err
		}
		tmpPr, resp, err := tp.gh.PullRequests.Get(ctx, tp.owner, tp.repo, *pr.Number)
		if err == nil {
			pr = tmpPr
		} else {
			tp.showGHError(err, resp)
		}
	}
	b, _ := json.Marshal(pr)
	tp.setOutput("pull_request", string(b))
	return nil
}
