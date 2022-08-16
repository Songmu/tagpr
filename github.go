package rcpr

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Songmu/gitconfig"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

func client(ctx context.Context, token, host string) (*github.Client, error) {
	if token == "" {
		var err error
		token, err = gitconfig.GitHubToken(host)
		if err != nil {
			return nil, err
		}
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauthClient := oauth2.NewClient(ctx, ts)
	client := github.NewClient(oauthClient)

	if host != "" {
		if host == "github.com" {
			host = "https://api.github.com"
		} else {
			// ref. https://github.com/google/go-github/issues/958
			host = fmt.Sprintf("https://%s/api/v3/", host)
		}
		u, err := url.Parse(host)
		if err != nil {
			return nil, err
		}
		client.BaseURL = u
	}
	return client, nil
}
