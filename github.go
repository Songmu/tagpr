package tagpr

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v83/github"
	"golang.org/x/oauth2"
)

func ghClient(ctx context.Context, token, host string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauthClient := oauth2.NewClient(ctx, ts)
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(oauthClient.Transport)
	if err != nil {
		return nil, err
	}
	client := github.NewClient(rateLimiter)

	fqdn := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		fqdn = h
	}
	if fqdn != "github.com" {
		if strings.HasSuffix(fqdn, ".ghe.com") {
			// for GitHub Enterprise Cloud
			// ref. https://docs.github.com/en/enterprise-cloud@latest/rest/using-the-rest-api/getting-started-with-the-rest-api
			host = fmt.Sprintf("https://api.%s", host)
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
