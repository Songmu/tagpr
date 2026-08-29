# tagpr ドキュメント

tagpr は、リリース内容をプルリクエストで準備し、作成されたバージョンタグをプロジェクト固有のリリースフローの起点とするカスタム GitHub Action です。

リリース前の変更をまとめたリリースプルリクエストを自動作成し、それがマージされると、マージコミットに自動でタグを付けます。デフォルトでは、GitHub Release の作成と changelog の生成も行います。

タグを付ける前の準備を自動化することで、リリース内容を可視化し、レビュー可能にします。また、リリース時に必要な変更を自動化し、作業負荷や属人性、ミスを減らします。

![リリース PR ブランチはリリースブランチから分岐し、タグ付きのリリースコミットでマージされる](images/release-flow.png)

## まずはこちら

- [はじめに](getting-started.md) では、ワークフローの追加から最初のリリースプルリクエストのマージまでの手順を説明します。
- [既存プロジェクトへの tagpr 導入](guides/adopting-tagpr.md) では、すでにあるプロジェクトに tagpr を導入する方法を説明します。
- [リリースフローと設計](concepts/release-flow.md) では、リリースプルリクエストのライフサイクルと、その背後にある考え方を説明します。
- [バージョンとラベルのルール](guides/versioning.md) では、SemVer の提案、カスタムラベル、バージョンファイルの優先順位、CalVer の動作を説明します。
- [Changelog と GitHub Releases](guides/changelog-and-releases.md) では、tagpr が GitHub の生成リリースノートと `.github/release.yml` を使う方法を説明します。
- [リリース準備コマンド](guides/release-commands.md) では、リリース前にプロジェクト固有のファイル変更を自動化する方法を説明します。
- [タグ付けとリリース](guides/tag-and-release.md) では、タグを起点としてビルド、パッケージ化、公開、デプロイなどのプロジェクト固有のリリース処理を実行する方法を説明します。
- [Immutable Releases の活用と連携](guides/immutable-releases.md) では、tagpr と後続のリリースワークフローを連携し、公開前にアセットを添付する方法を説明します。
- [設定インデックス](reference/configuration.md) では、利用可能な設定を目的別にまとめ、完全なリファレンスへのリンクを示します。
- [リリースプルリクエストテンプレート](reference/templates.md) では、タイトルと本文のカスタマイズ方法を説明します。

## よくある目的

| 目的                                | ドキュメント                                                                                    |
| --------------------------------- | ----------------------------------------------------------------------------------------- |
| tagpr をインストールする                   | [はじめに](getting-started.md)                                                                |
| 既存プロジェクトに tagpr を導入する             | [導入ガイド](guides/adopting-tagpr.md)                                                         |
| バージョンファイルなしでリリースする                | [タグのみのリリース](getting-started.md#tag-only-releases)                                         |
| major または minor バージョンを選ぶ          | [バージョンとラベルのルール](guides/versioning.md)                                                     |
| Calendar Versioning を使う           | [設定インデックス](reference/configuration.md#version-selection)                                  |
| monorepo のプロジェクトをリリースする           | [設定インデックス](reference/configuration.md#monorepos)                                          |
| changelog のカテゴリをカスタマイズする          | [Changelog と GitHub Releases](guides/changelog-and-releases.md#customize-generated-notes) |
| リリースプルリクエストをカスタマイズする              | [リリースプルリクエストテンプレート](reference/templates.md)                                               |
| プロジェクト固有のリリース変更を実行する              | [リリース準備コマンド](guides/release-commands.md)                                                  |
| 同じワークフローでリリースする                   | [タグ付けとリリース](guides/tag-and-release.md#publish-in-the-same-workflow)                     |
| 別のリリースワークフローを起動する                | [タグ付けとリリース](guides/tag-and-release.md#trigger-a-separate-workflow)                      |
| immutable release でアセットを公開する      | [Immutable Releases の活用と連携](guides/immutable-releases.md)                              |
| GoReleaser で tagpr の draft を再利用する | [Immutable Releases の活用と連携](guides/immutable-releases.md#reuse-the-draft-with-goreleaser) |
| セットアップの問題を診断する                    | [README のトラブルシューティング](../README.md#troubleshooting)                                       |

## リファレンス

完全な設定、GitHub Action の入力、出力、環境変数、GitHub Enterprise の手順は現在も
[README](../README.md#configuration-reference) にあります。同等の独立したリファレンスページが完成するまではそこに残るため、ドキュメントサイトへの移行中も情報が失われることはありません。
