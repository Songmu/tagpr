# リリース後の公開

tagpr はバージョンタグを作成し、下流のステップが利用できる出力を提供します。tagpr ワークフロー内で公開するか、別のワークフローを起動するかを選んでください。

下流の処理で GitHub Release のアセットを追加し、リポジトリが immutable release を使う場合は、すべてのアセットを添付してから公開しなければなりません。`tagpr.release = draft` と `tagpr.release = false` の連携パターンは [不変な GitHub Releases](immutable-releases.md) を参照してください。

## `GITHUB_TOKEN` の制約 {#github_token-constraints}

リポジトリの `GITHUB_TOKEN` は、GitHub がワークフロー実行ごとに自動作成するため、tagpr で使う最も簡単な認証情報です。ただし、`GITHUB_TOKEN` で作成されたイベントは[通常、別のワークフロー実行を開始しません][github-token-trigger]。これは tagpr に対して次の 2 箇所に影響します。

- tagpr が作成したタグは、`on.push.tags` で設定したワークフローを起動しない。
- リリース PR 用の対象となる `pull_request` ワークフローはキューに入るが、[書き込み権限を持つユーザーが承認する][bot-pr-approval]まで実行されない。

tagpr がタグを作成した後に公開またはデプロイを自動実行する方法は 2 つあります。

| 構成 | 利点 | トレードオフ |
| --- | --- | --- |
| tagpr ワークフロー内で公開する | 追加の認証情報なしで `GITHUB_TOKEN` を使える | リリース PR ワークフローに承認が必要で、公開処理が tagpr のワークフロー権限と環境を共有する |
| 別のタグワークフローを起動する | リリースの責任を分離でき、タグとリリース PR のワークフローを自動実行できる | ワークフローを起動できるトークンが必要 |

## 同じワークフローで公開する {#publish-in-the-same-workflow}

`tag` 出力は tagpr がタグを作成した場合にのみ空でなくなります。同じワークフロー内の公開またはデプロイステップの条件に使います。

```yaml
- uses: actions/checkout@v6
  with:
    persist-credentials: false
- id: tagpr
  uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
- name: Publish
  if: steps.tagpr.outputs.tag != ''
  uses: ./.github/actions/release
  with:
    tag: ${{ steps.tagpr.outputs.tag }}
    token: ${{ secrets.GITHUB_TOKEN }}
```

この構成では GitHub App も personal access token も必要ありません。公開ロジックをスクリプトまたはローカルの composite action に置けば、tagpr と公開処理がワークフローを共有していても結合を抑えられます。完全な例は [Songmu/ecschedule's tagpr workflow][ecschedule-tagpr] を参照してください。

利用可能なその他の出力は次のとおりです。

- `pull_request`: リリースプルリクエストを説明する JSON。
- `base_tag`: 比較の基準に使った前回のタグ。最初のリリースでは空の値。

## 別のワークフローを起動する {#trigger-a-separate-workflow}

公開またはデプロイを `on.push.tags` で設定したワークフローに置くには、次のようにします。

```yaml
on:
  push:
    tags:
    - "v*"
```

タグでそのワークフローを起動できるよう、`GITHUB_TOKEN` 以外のトークンを tagpr に渡します。personal access token も使えますが、[`actions/create-github-app-token`][create-app-token] で作成する短期間の GitHub App インストールトークンを推奨します。

GitHub App は次の権限でリポジトリにインストールする必要があります。

- Contents: Read and write
- Pull requests: Read and write
- Issues: Read-only

App の作成、インストール、認証情報の保存については、`actions/create-github-app-token` のドキュメントで説明しています。設定後、トークンを生成し、checkout と tagpr の両方に使います。

```yaml
- name: Generate token
  id: app-token
  uses: actions/create-github-app-token@v3
  with:
    client-id: ${{ secrets.CLIENT_ID }}
    private-key: ${{ secrets.PRIVATE_KEY }}
    permission-contents: write
    permission-pull-requests: write
    permission-issues: read

- uses: actions/checkout@v6
  with:
    token: ${{ steps.app-token.outputs.token }}
    persist-credentials: false

- uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
```

このインストールトークンで作成されたタグとリリース PR の更新は、`GITHUB_TOKEN` の制約なしに、それぞれのワークフローを起動できます。

**Allow GitHub Actions to create and approve pull requests** のリポジトリ設定は `GITHUB_TOKEN` を制御します。GitHub App トークンは代わりに App の権限で管理されます。

## 公開を復旧可能にする

どのワークフロー構成を選んでも、公開処理が明示的なタグを受け取れるようにしてください。これにより、別のリリースタグを作成せずに、失敗したリリースジョブを再実行したり手動で起動したりできます。

たとえば、パッケージ化とアップロードのロジックをスクリプトまたはローカルの composite action に置きます。

```yaml
- name: Publish
  run: ./.github/scripts/publish "${{ inputs.tag }}"
```

tagpr ワークフローと復旧ワークフローの両方から同じ処理を呼び出せます。

## セキュリティ上の考慮事項

- 長期間有効な personal access token より、短期間の GitHub App インストールトークンを優先する。
- tagpr と公開処理に必要な権限だけを付与する。
- checkout で認証情報がローカルの Git 設定に保持されないよう、`persist-credentials: false` を維持する。
- リポジトリのサプライチェーンポリシーに従って、サードパーティ Action を固定する。

Action の完全な出力リファレンスは、[README](../../README.md#outputs) を参照してください。

[bot-pr-approval]: https://github.blog/changelog/2026-06-11-bot-created-pull-requests-can-run-workflows-if-approved/
[create-app-token]: https://github.com/actions/create-github-app-token
[ecschedule-tagpr]: https://github.com/Songmu/ecschedule/blob/main/.github/workflows/tagpr.yaml
[github-token-trigger]: https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-when-your-workflow-runs/triggering-a-workflow
