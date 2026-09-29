package tagpr

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNewRateLimitClientDoesNotEnablePrimaryLimiter(t *testing.T) {
	t.Parallel()

	requests := 0
	client := newRateLimitClient(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header: http.Header{
					"X-Ratelimit-Remaining": []string{"0"},
					"X-Ratelimit-Reset":     []string{"4102444800"},
				},
				Body: io.NopCloser(strings.NewReader(
					`{"message":"You have exceeded a secondary rate limit"}`,
				)),
				Request: req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}))

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/Songmu/tagpr", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	resp.Body.Close()

	req, err = http.NewRequest(http.MethodGet, "https://api.github.com/repos/Songmu/tagpr", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	resp.Body.Close()

	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
