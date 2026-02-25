module github.com/Songmu/tagpr

go 1.25.0

require (
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/Songmu/gitconfig v0.2.2
	github.com/Songmu/gitsemvers v0.1.0
	github.com/Songmu/tagpr/gh2changelog v0.7.1
	github.com/gofri/go-github-ratelimit v1.1.1
	github.com/google/go-github/v83 v83.0.0
	github.com/k1LoW/calver v1.0.1
	github.com/saracen/walker v0.1.4
	golang.org/x/oauth2 v0.35.0
)

require (
	github.com/cli/go-gh/v2 v2.13.0 // indirect
	github.com/cli/safeexec v1.0.1 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/snabb/isoweek v1.0.3 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/Songmu/tagpr/gh2changelog => ./gh2changelog
