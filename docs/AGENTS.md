# Documentation contributor instructions

## English and Japanese documentation

- Keep public documentation pages in English (`*.md`) and Japanese (`*.ja.md`)
  synchronized in the same change.
- When updating a page, update its counterpart's explanations, examples,
  configuration paths, and links as needed.
- Add both language versions when creating a public page. Preserve explicit
  heading anchors used by links across languages.

## Site generation

- `docs/site/build.sh` generates the site and checks that every public page has
  both language versions.
- `AGENTS.md` files contain contributor instructions, not public documentation.
  Keep them excluded from page discovery and translation checks at every depth;
  they do not need Japanese counterparts.
- Run `docs/site/build.sh` with a temporary output directory after changing
  documentation or site generation.
