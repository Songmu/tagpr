# はじめに

このガイドでは、`main` をリリースブランチとし、Semantic Versioning を使うリポジトリに tagpr を導入します。Calendar Versioning、monorepo、メンテナンスブランチでも、追加設定は必要ですが基本的なリリースフローは同じです。

リポジトリにすでに公開済みのバージョン、changelog、またはリリース自動化がある場合は、先に [既存プロジェクトへの tagpr 導入](guides/adopting-tagpr.md) を読んでください。

## 前提条件

- リポジトリで GitHub Actions を使っている。
- リリース元のブランチが 1 つ指定されており、通常は `main` である。
- ワークフローにリポジトリの内容とプルリクエストを書き込む権限がある。
- デフォルト設定では、リリースタグが `v1.2.3` のような SemVer に従う。

## ワークフローを追加する {#add-the-workflow}

`.github/workflows/tagpr.yml` を作成します。

```yaml
name: tagpr
on:
  push:
    branches: ["main"]
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: write
  issues: read

jobs:
  tagpr:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v6
      with:
        persist-credentials: false
    - uses: Songmu/tagpr@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

明示的な権限により、tagpr はリリース PR ブランチの push、リリースプルリクエストの作成と更新、マージ済みプルリクエストの調査、タグと GitHub Release の作成を行えます。`persist-credentials: false` により、tagpr は checkout が保持した認証情報ではなく、環境経由で渡されたトークンを Git 操作に使います。

> [!CAUTION]
> デフォルトの `GITHUB_TOKEN` では、tagpr が作成したタグによって別のワークフローが起動することはありません。また、tagpr が作成または更新したリリースプルリクエストを対象とする `pull_request` ワークフローは承認待ち状態で作成され、実行には書き込み権限を持つユーザーの承認が必要です。詳細と別のワークフロー構成については、[タグ付けとリリース](guides/tag-and-release.md) を参照してください。

## プルリクエスト作成を有効にする

リポジトリで **Settings > Actions > General > Workflow permissions** を開き、**Allow GitHub Actions to create and approve pull requests** を有効にします。

tagpr が `GITHUB_TOKEN` を使う場合、この設定も必要です。GitHub App トークンの場合は App の権限によって管理されます。

## 初めて tagpr を実行する

ワークフローをコミットして `main` に push します。最初の実行では次の処理が行われます。

1. 最新の SemVer タグを探します。存在しない場合、tagpr は最初のコミット以降の変更を含めます。
2. `.tagpr` がなければ作成し、`versionFile` の設定がない場合は有力なバージョンファイルを検出します。
3. `.github/release.yml` と `.github/release.yaml` のどちらも存在しない場合、`.github/release.yml` を作成します。
4. 提案されたバージョンと changelog を含むリリースプルリクエストを作成します。

生成された `.tagpr` ファイルを確認します。`tagpr.releaseBranch` が正しく、`tagpr.versionFile` にリリース時に変更すべきすべてのファイルが指定されていることを確認してください。

例:

```ini
[tagpr]
    releaseBranch = main
    versionFile = version.go,action.yml
    vPrefix = true
```

`versionFile` には、`package.json` や `Cargo.toml` など、プロジェクトのバージョンが記述されたファイルを指定します。tagpr はこれらのファイルに含まれるバージョンを自動更新し、リリースプルリクエストに反映します。カンマ区切りで複数のファイルを指定できるため、リリース時にまとめて更新したいファイルをすべて列挙してください。

> [!CAUTION]
> バージョンの自動更新では、各ファイル内で現在のバージョンに最初に一致した箇所だけを置換します。意図した箇所が更新されることを、最初のリリースプルリクエストで確認してください。

また、tagpr は `releaseBranch` にマージされた各プルリクエストを changelog の項目として扱います。

### タグのみのリリース {#tag-only-releases}

プロジェクトがバージョンファイルを持たない場合は、次のように指定します。

```ini
[tagpr]
    versionFile = -
```

tagpr は引き続き changelog を準備し、リリースプルリクエストを作成し、マージされたコミットにタグを付けます。

## 最初のリリースを準備する

生成されたリリースプルリクエストは、その後の `main` への push を追跡します。リリースを開始する準備ができるまで開いたままにしてください。

マージ前に次のことができます。

- 提案されたバージョンと changelog を確認する
- リリース PR ブランチ（`tagpr-from-*`）にプロジェクト固有のリリース変更を直接コミットする
- リリース対象のプルリクエストのタイトルを編集し、changelog に表示される内容を調整する
- リリース対象のプルリクエストにラベルを付け、changelog のカテゴリを調整する
- `tagpr:minor` または `tagpr:major` を追加し、tagpr が提案する次のバージョンを変更する
- 設定したバージョンファイルを編集して次バージョンを明示的に指定する

`main` ブランチに新しいコミットを追加せずにリリースプルリクエストを更新したい場合は、`workflow_dispatch` トリガーから tagpr を手動で再実行できます。

## 次バージョンの選定

デフォルトの提案は patch リリースです。前回のリリース後にマージされた通常のプルリクエストに `major` や `minor` のラベルがある場合、tagpr はリリースプルリクエストに `tagpr:major` や `tagpr:minor` のラベルを追加します。判定に使うラベル名は、`tagpr.majorLabels` と `tagpr.minorLabels` で変更できます。

Dependabot のプルリクエストラベルは、プロジェクトのリリースではなく依存関係のバージョン変更を表すため無視されます。

リリースプルリクエスト上での次バージョン決定ロジックは、次のようになります。

- `tagpr:major` または `tagpr/major` は major の更新を選ぶ。
- `tagpr:minor` または `tagpr/minor` は minor の更新を選ぶ。
- バージョンラベルがない場合は patch の更新を選ぶ。
- 編集されたバージョンファイルの内容がラベルより優先される。

カスタムラベルの対応付けと完全な優先順位については、[バージョンとラベルのルール](guides/versioning.md) を参照してください。

## タグ付けとリリース開始

リリースプルリクエストをマージします。マージによって `main` が進み、tagpr が再度実行されます。tagpr はマージされたリリースプルリクエストを認識し、マージされたコミットにタグを付けます。そのタグを起点として、プロジェクト固有のリリースフローが始まります。`tagpr.release` で無効にしていない限り GitHub Release も作成します。

タグを使ってビルド、パッケージ化、公開、デプロイなどを実行するには、[タグ付けとリリース](guides/tag-and-release.md) に進んでください。

## 次のステップ

- [リリースフローと設計](concepts/release-flow.md) を確認する。
- 追加設定を [設定インデックス](reference/configuration.md) で探す。
- ワークフローがリリースプルリクエストを作成または更新できない場合は、[README のトラブルシューティングガイド](../README.md#troubleshooting) を使う。
