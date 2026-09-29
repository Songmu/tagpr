package tagpr

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v92/github"
	"golang.org/x/oauth2"
)

func ghClient(ctx context.Context, token, host string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauthClient := oauth2.NewClient(ctx, ts)
	rateLimiter := newRateLimitClient(oauthClient.Transport)
	fqdn := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		fqdn = h
	}
	opts := []github.ClientOptionsFunc{github.WithHTTPClient(rateLimiter)}
	if fqdn != "github.com" {
		if strings.HasSuffix(fqdn, ".ghe.com") {
			// for GitHub Enterprise Cloud
			// ref. https://docs.github.com/en/enterprise-cloud@latest/rest/using-the-rest-api/getting-started-with-the-rest-api
			host = fmt.Sprintf("https://api.%s/", host)
		} else {
			// ref. https://github.com/google/go-github/issues/958
			host = fmt.Sprintf("https://%s/api/v3/", host)
		}
		opts = append(opts, github.WithURLs(&host, nil))
	}
	client, err := github.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func newRateLimitClient(base http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: github_ratelimit.NewSecondaryLimiter(base),
	}
}
