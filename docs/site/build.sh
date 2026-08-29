#!/usr/bin/env bash
set -euo pipefail

site_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
docs_dir="$(cd "$site_dir/.." && pwd)"
repo_dir="$(cd "$docs_dir/.." && pwd)"
output_dir="${1:-"$repo_dir/site"}"

if [[ "$output_dir" != /* ]]; then
  output_dir="$PWD/$output_dir"
fi

mkdir -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd -P)"
case "$output_dir" in
  /|"$repo_dir"|"$docs_dir"|"$site_dir")
    echo "refusing to use unsafe output directory: $output_dir" >&2
    exit 1
    ;;
esac

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

mkdir -p "$work_dir/content" "$work_dir/static"
cp "$site_dir/hugo.yaml" "$site_dir/go.mod" "$site_dir/go.sum" "$work_dir/"
cp -R "$site_dir/layouts" "$work_dir/layouts"
cp -R "$docs_dir/images" "$work_dir/static/images"

manifest="$work_dir/translations.tsv"
: > "$manifest"

while IFS= read -r -d '' source; do
  relative="${source#"$docs_dir/"}"
  language="en"
  logical="$relative"
  case "$relative" in
    *.ja.md)
      language="ja"
      logical="${relative%.ja.md}.md"
      ;;
    *.md)
      filename="${relative##*/}"
      stem="${filename%.md}"
      if [[ "$stem" == *.* ]]; then
        echo "unsupported or malformed documentation language suffix: $relative" >&2
        exit 1
      fi
      ;;
  esac

  printf '%s\t%s\t%s\n' "$language" "$logical" "$relative" >> "$manifest"

  if [[ "$logical" == "index.md" ]]; then
    if [[ "$language" == "en" ]]; then
      target="$work_dir/content/_index.md"
    else
      target="$work_dir/content/_index.$language.md"
    fi
  else
    target="$work_dir/content/$relative"
  fi

  IFS= read -r heading < "$source" || heading=""
  if [[ "$heading" != "# "* ]]; then
    echo "documentation page must start with a level-one heading: $relative" >&2
    exit 1
  fi

  title="${heading#\# }"
  title="${title//\\/\\\\}"
  title="${title//\"/\\\"}"

  weight=""
  case "$logical" in
    getting-started.md) weight=10 ;;
    guides/adopting-tagpr.md) weight=10 ;;
    guides/versioning.md) weight=20 ;;
    guides/changelog-and-releases.md) weight=30 ;;
    guides/release-commands.md) weight=40 ;;
    guides/tag-and-release.md) weight=50 ;;
    guides/immutable-releases.md) weight=60 ;;
    concepts/release-flow.md) weight=10 ;;
    reference/configuration.md) weight=10 ;;
    reference/templates.md) weight=20 ;;
  esac

  mkdir -p "$(dirname "$target")"
  {
    printf '%s\n' '---'
    printf 'title: "%s"\n' "$title"
    if [[ -n "$weight" ]]; then
      printf 'weight: %s\n' "$weight"
    fi
    if [[ "$logical" == "guides/tag-and-release.md" ]]; then
      printf '%s\n' 'aliases:' '  - publish-after-release'
    fi
    if [[ "$logical" == "index.md" ]]; then
      printf '%s\n' 'type: docs' 'cascade:' '  type: docs'
    fi
    printf '%s\n\n' '---'
    tail -n +2 "$source"
  } > "$target"
done < <(
  find "$docs_dir" \
    -path "$site_dir" -prune -o \
    -type f -name '*.md' -print0
)

duplicates="$(
  cut -f 1,2 "$manifest" |
    LC_ALL=C sort |
    uniq -d
)"
if [[ -n "$duplicates" ]]; then
  echo "duplicate documentation translations:" >&2
  echo "$duplicates" >&2
  exit 1
fi

english_pages="$work_dir/english-pages"
japanese_pages="$work_dir/japanese-pages"
awk -F '	' '$1 == "en" { print $2 }' "$manifest" | LC_ALL=C sort > "$english_pages"
awk -F '	' '$1 == "ja" { print $2 }' "$manifest" | LC_ALL=C sort > "$japanese_pages"

missing_japanese="$(comm -23 "$english_pages" "$japanese_pages")"
if [[ -n "$missing_japanese" ]]; then
  echo "missing Japanese documentation translations:" >&2
  echo "$missing_japanese" >&2
  exit 1
fi

orphaned_japanese="$(comm -13 "$english_pages" "$japanese_pages")"
if [[ -n "$orphaned_japanese" ]]; then
  echo "Japanese documentation translations without English sources:" >&2
  echo "$orphaned_japanese" >&2
  exit 1
fi

write_section() {
  local path="$1"
  local title="$2"
  local weight="$3"
  local language="${4:-en}"
  local suffix=""
  if [[ "$language" != "en" ]]; then
    suffix=".$language"
  fi
  cat > "$work_dir/content/$path/_index$suffix.md" <<EOF
---
title: "$title"
weight: $weight
---
EOF
}

write_section guides Guides 20
write_section guides ガイド 20 ja
write_section concepts Concepts 30
write_section concepts コンセプト 30 ja
write_section reference Reference 40
write_section reference リファレンス 40 ja

hugo \
  --source "$work_dir" \
  --destination "$output_dir" \
  --cleanDestinationDir \
  --gc \
  --minify \
  --panicOnWarning \
  --printPathWarnings
