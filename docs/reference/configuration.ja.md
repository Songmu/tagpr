# 設定インデックス

tagpr は Git の設定形式で `.tagpr` を読み取ります。すべての設定には対応する `TAGPR_*` 環境変数もあり、環境変数が設定ファイルより優先されます。

```ini
[tagpr]
    releaseBranch = main
    versionFile = version.go
    changelog = true
    release = draft
```

項目ごとの完全な説明とデフォルト値は現在、[README の設定リファレンス](../../README.md#configuration-reference) にあります。このページでは設定を目的別にまとめ、関係するオプションをすばやく見つけられるようにしています。

## パスの解決

相対パスは `.tagpr` を含むディレクトリではなく、tagpr の作業ディレクトリから解決されます。GitHub Action は tagpr の実行前に `GITHUB_WORKSPACE` へ移動するため、設定のパスはリポジトリルートからの相対パスです。

これは `versionFile`、`changelogFile`、`releaseYAMLPath`、`template` に適用されます。設定したコマンドもリポジトリルートから実行され、自動的なバージョンファイル検出もそこからスキャンします。

## リリース対象

| 設定 | 目的 |
| --- | --- |
| `tagpr.releaseBranch` | リリースプルリクエストをマージするリリースブランチ |
| `tagpr.fixedMajorVersion` | 1 つの major リリース系列にあるメンテナンスブランチを対象にする |

## バージョン選択 {#version-selection}

| 設定 | 目的 |
| --- | --- |
| `tagpr.versionFile` | バージョンファイルのパス、自動検出、またはタグのみのモード |
| `tagpr.vPrefix` | SemVer タグに `v` を追加する |
| `tagpr.majorLabels` | major リリースを意味する、マージ済みプルリクエストのラベル |
| `tagpr.minorLabels` | minor リリースを意味する、マージ済みプルリクエストのラベル |
| `tagpr.calendarVersioning` | CalVer を有効にし、その形式を選ぶ |

SemVer、バージョンファイル、CalVer の完全な優先順位ルールは、[バージョンとラベルのルール](../guides/versioning.md) を参照してください。

## 生成ファイルとコマンド

| 設定 | 目的 |
| --- | --- |
| `tagpr.changelog` | changelog の更新を有効または無効にする |
| `tagpr.changelogFile` | changelog のパスを選ぶ |
| `tagpr.releaseYAMLPath` | 生成するリリースノート設定を選ぶ |
| `tagpr.command` | バージョンファイルの更新前にコマンドを実行する |
| `tagpr.postVersionCommand` | バージョンファイルの更新後にコマンドを実行する |

コマンドには `TAGPR_CURRENT_VERSION` と `TAGPR_NEXT_VERSION` が渡されます。実行順序、例、ファイルの扱いの詳細は、[リリース準備コマンド](../guides/release-commands.md) を参照してください。

## リリースプルリクエスト

| 設定 | 目的 |
| --- | --- |
| `tagpr.template` | プルリクエストに Go テンプレートファイルを使う |
| `tagpr.templateText` | インラインの Go テンプレートを使う |
| `tagpr.commitPrefix` | tagpr のコミットのプレフィックスをカスタマイズする |

出力形式、利用可能な変数、例については、[リリースプルリクエストテンプレート](templates.md) を参照してください。

## GitHub Release

| 設定 | 目的 |
| --- | --- |
| `tagpr.release` | 公開済みリリース、draft、または GitHub Release なしを選ぶ |

後続ワークフローが公開前にアセットを添付する必要がある場合は、[Immutable Releases の活用と連携](../guides/immutable-releases.md) を参照してください。

## Monorepo {#monorepos}

`tagpr.tagPrefix` を使って、独立してリリースする各プロジェクトに専用のタグ名前空間を与えます。設定自体はプロジェクトのディレクトリに置けます。

```yaml
- uses: Songmu/tagpr@v1
  with:
    config: tools/.tagpr
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

そのファイル内のパスもリポジトリルートからの相対パスです。バージョンファイル、changelog、リリースノート設定も対象を限定してください。

```ini
# tools/.tagpr
[tagpr]
    tagPrefix = tools
    versionFile = tools/package.json
    changelogFile = tools/CHANGELOG.md
    releaseYAMLPath = tools/.github/release.yml
```

この設定では `tools/v1.2.3` のようなタグが作成されます。

## Action の設定

GitHub Action には 2 つの入力があります。

- `config` は `.tagpr` 以外の設定ファイルを選ぶ。
- `version` は Action がインストールする tagpr 実行可能ファイルのバージョンを選ぶ。

`tag`、`pull_request`、`base_tag` の出力を提供します。使用方法は [タグ付けとリリース](../guides/tag-and-release.md) を参照してください。
