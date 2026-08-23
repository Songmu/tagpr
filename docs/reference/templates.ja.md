# リリースプルリクエストテンプレート

tagpr は Go の text template を使って、リリースプルリクエスト全体を描画します。デフォルトテンプレートは、ファイルまたはインラインテキストで置き換えられます。

## 出力形式

描画結果の空でない最初の行がプルリクエストのタイトルになります。描画結果の残りすべてがプルリクエストの本文になります。

たとえば、次のテンプレート:

```gotemplate
Release {{.NextVersion}}

Merging this pull request will create {{.NextVersion}}.

{{.Changelog}}
```

は、タイトルに `Release v1.2.3` を生成し、残りの Markdown を本文として使います。

## 利用可能な変数

| 変数 | 説明 | 例 |
| --- | --- | --- |
| `.NextVersion` | 有効時に `v` を含み、`tagPrefix` を除いた提案バージョン | `v1.2.3` |
| `.Branch` | リリースプルリクエストに使うブランチ | `tagpr-from-v1.2.2` |
| `.Changelog` | GitHub が返す生成リリースノート | `## What's Changed ...` |
| `.TagPrefix` | 末尾のスラッシュを除いた、正規化済み monorepo タグプレフィックス | `tools` |

同じテンプレートを複数のプロジェクトで使う場合は `.TagPrefix` を使います。

```gotemplate
{{if .TagPrefix}}[{{.TagPrefix}}] {{end}}Release {{.NextVersion}}

{{.Changelog}}
```

## テンプレートファイル

テンプレートのパスを `tagpr.template` に設定します。

```ini
[tagpr]
    template = .github/tagpr-template.md
```

GitHub Action を使う場合、`.tagpr` 設定がサブディレクトリにあっても、パスはリポジトリルートからの相対パスです。

monorepo の場合:

```ini
# tools/.tagpr
[tagpr]
    tagPrefix = tools
    template = tools/.github/tagpr-template.md
```

## インラインテンプレート

テンプレートを直接指定するには、`tagpr.templateText` または `TAGPR_TEMPLATE_TEXT` を設定します。GitHub Actions では複数行の環境変数が便利です。

```yaml
- uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAGPR_TEMPLATE_TEXT: |
      Release {{.NextVersion}}

      This release is prepared from {{.Branch}}.

      {{.Changelog}}
```

`tagpr.template` と `tagpr.templateText` の両方が設定されている場合は、テンプレートファイルが優先されます。

## Changelog の配置

GitHub の生成リリースノートを表示する場所に `{{.Changelog}}` を含めます。これを省略するとリリースプルリクエスト本文からそのノートが削除されますが、changelog ファイルの更新や GitHub Release の生成は無効になりません。

ノートの生成方法は、[Changelog と GitHub Releases](../guides/changelog-and-releases.md) を参照してください。

## 無効なテンプレート

tagpr が設定されたテンプレートを解析できない場合、または描画に失敗した場合は、エラーをログに記録し、組み込みのデフォルトテンプレートにフォールバックします。
