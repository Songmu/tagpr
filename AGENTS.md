# Contributor instructions

## Hugo site dependencies

`docs/site` is a Hugo module, not a Go source module.

- Never run `go mod tidy` in `docs/site`. It removes Hugo theme dependencies,
  such as Hextra, because they are not Go packages.
- Use `make docs-deps` or run `hugo mod tidy` from `docs/site` instead.
- Running `go mod tidy` at the repository root is safe; it does not traverse
  into the nested `docs/site` module.

## GitHub Actions dependencies

- Install the lockfile tool with
  `gh extension install github/gh-actions-lock --pin v0.2.0`.
- After adding or changing workflow dependencies or local actions, run
  `gh actions-lock` from the repository root and commit the generated
  `.github/workflows/actions.lock` together with the workflow changes.
- Never edit the lockfile by hand. Run `gh actions-lock --verify` to check it.
- Dependabot maintains the lockfile when updating existing GitHub Actions
  dependencies.
