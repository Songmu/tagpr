tagpr
=======

[![Test Status](https://github.com/Songmu/tagpr/workflows/test/badge.svg?branch=main)][actions]
[![Coverage Status](https://codecov.io/gh/Songmu/tagpr/branch/main/graph/badge.svg)][codecov]
[![MIT License](https://img.shields.io/github/license/Songmu/tagpr)][license]
[![PkgGoDev](https://pkg.go.dev/badge/github.com/Songmu/tagpr)][PkgGoDev]

[actions]: https://github.com/Songmu/tagpr/actions?workflow=test
[codecov]: https://codecov.io/gh/Songmu/tagpr
[license]: https://github.com/Songmu/tagpr/blob/main/LICENSE
[PkgGoDev]: https://pkg.go.dev/github.com/Songmu/tagpr

tagpr short description

## Synopsis

```go
// simple usage here
```

## Description

## Installation

```console
# Install the latest version. (Install it into ./bin/ by default).
% curl -sfL https://raw.githubusercontent.com/Songmu/tagpr/main/install.sh | sh -s

# Specify installation directory ($(go env GOPATH)/bin/) and version.
% curl -sfL https://raw.githubusercontent.com/Songmu/tagpr/main/install.sh | sh -s -- -b $(go env GOPATH)/bin [vX.Y.Z]

# In alpine linux (as it does not come with curl by default)
% wget -O - -q https://raw.githubusercontent.com/Songmu/tagpr/main/install.sh | sh -s [vX.Y.Z]

# go install
% go install github.com/Songmu/tagpr/cmd/tagpr@latest
```

## GitHub Actions

Action Songmu/tagpr@main installs tagpr binary for Linux into /usr/local/bin and run it.

```yaml
name: tagpr
on:
  push:
    branches: ["main"]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: Songmu/tagpr@main
```

## Author

[Songmu](https://github.com/Songmu)
