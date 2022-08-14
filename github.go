package rcpr

import (
	"context"
	"net/url"

	"github.com/Songmu/gitconfig"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

func client(ctx context.Context, token, baseURL string) (*github.Client, error) {
	if token == "" {
		var err error
		token, err = gitconfig.GitHubToken(baseURL)
		if err != nil {
			return nil, err
		}
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauthClient := oauth2.NewClient(ctx, ts)
	client := github.NewClient(oauthClient)

	if baseURL != "" {
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}
		client.BaseURL = u
	}
	return client, nil
}
