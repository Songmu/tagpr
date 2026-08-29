# Changelog と GitHub Releases

tagpr はリリースノートの生成を GitHub に委任します。独自のプルリクエスト分類ルールを実装するのではなく、[Generate release notes API][github-generate-release-notes-api] を使います。

これにより、Git の生の履歴をコピーするより有用な出発点が得られます。プルリクエストのタイトルはユーザーに見える変更を説明し、ラベルでグループ化でき、メンテナーはリリース前に結果をレビューできます。tagpr は生成された Markdown を [Keep a Changelog][keep-a-changelog] 形式の changelog エントリに変換します。

## 生成フロー

tagpr はリリースプルリクエストを準備するとき、次の情報を GitHub に渡します。

- 提案された次のタグ。
- 存在する場合は前回のリリースタグ。
- 設定されたリリースブランチ。
- リリースノート設定のパス。

GitHub は 2 つのバージョン間にあるプルリクエストとコントリビューターの生成リリースノートを返します。tagpr はそのノートを changelog エントリに変換し、生成されたノートをリリースプルリクエストの本文に含めます。

リリースプルリクエストのマージ後、tagpr は最終タグ用にノートを再生成し、返されたタイトルと本文を GitHub Release の作成時に使います。リリース作成前にノートを生成することで、リリースプルリクエスト自身がリリース済みの変更として含まれることも防ぎます。

## 生成ノートをカスタマイズする {#customize-generated-notes}

GitHub はデフォルトで `.github/release.yml` または `.github/release.yaml` を読み取ります。このファイルでは次を定義できます。

- changelog のカテゴリと、プルリクエストをそこへ分類するラベル。
- リリースノートから除外するラベルと作成者。
- 分類されない変更に使うラベル。

完全なスキーマは GitHub の[自動生成リリースノートのドキュメント][github-generated-release-notes]を参照してください。

どちらのデフォルトファイルも存在しない場合、tagpr は初回実行時に次の最小設定を作成します。

```yaml
changelog:
  exclude:
    labels:
      - tagpr
```

この除外により、tagpr 自身のリリースプルリクエストが生成された変更一覧から除外されます。

## 別の設定パスを使う

monorepo の各プロジェクトに独自のリリースノートルールが必要な場合は、`tagpr.releaseYAMLPath` を設定します。

```ini
# tools/.tagpr
[tagpr]
    tagPrefix = tools
    changelogFile = tools/CHANGELOG.md
    releaseYAMLPath = tools/.github/release.yml
```

パスは `.tagpr` ファイルからではなく、リポジトリルートからの相対パスです。設定したファイルがリリースブランチに存在しない場合は、`tagpr.releaseYAMLPath` を設定する前に作成してコミットしてください。デフォルトの `.github/release.yml` とは異なり、カスタム設定パスが存在しない場合、リリースプルリクエストから自動的に用意することはできません。

## 制御ファイルとリリース作成

生成されたノートはいくつかの場所で使われますが、次の設定で tagpr が書き込む成果物を制御します。

- `tagpr.changelog = false` は changelog ファイルの作成または更新を停止する。
- `tagpr.changelogFile` はデフォルトの `CHANGELOG.md` から changelog のパスを変更する。
- `tagpr.release = true` は公開済み GitHub Release を作成する。
- `tagpr.release = draft` は draft の GitHub Release を作成する。
- `tagpr.release = false` は GitHub Release を作成せずにタグを作成する。

changelog ファイルの更新または GitHub Release の作成を無効にしても、リリースプルリクエストの本文では GitHub の生成ノートが使われます。

タグ付け後にリリースアセットをビルドする必要がある場合は、immutable release を有効にする前に [Immutable Releases の活用と連携](immutable-releases.md) を参照してください。tagpr に draft の準備を任せるタイミングと、別のワークフローにリリース作成を委任するタイミングを説明しています。

[github-generated-release-notes]: https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes
[github-generate-release-notes-api]: https://docs.github.com/en/rest/releases/releases#generate-release-notes-content-for-a-release
[keep-a-changelog]: https://keepachangelog.com/
