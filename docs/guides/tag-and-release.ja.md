# タグ付けとリリース

tagpr では、バージョンタグの作成をリリースフローの起点とします。タグ付け後のリリースフローでは、成果物のビルドやパッケージ化、パッケージや GitHub Release アセットの公開、アプリケーションのデプロイなどを実行できます。

タグは、リリース対象のソースとバージョンを一意に示します。後続のステップでは tagpr の `tag` 出力を利用でき、タグの push を契機として別のリリースワークフローを起動することもできます。

> [!NOTE]
> リポジトリで immutable release を有効にし、後続の処理で GitHub Release にアセットを追加する場合は、すべてのアセットを添付してから公開する必要があります。`tagpr.release = draft` と `tagpr.release = false` の連携パターンは、[Immutable Releases の活用と連携](immutable-releases.md) を参照してください。

## `GITHUB_TOKEN` の制約 {#github_token-constraints}

リポジトリの `GITHUB_TOKEN` は、GitHub がワークフロー実行ごとに自動作成するため、tagpr で使う最も簡単な認証情報です。ただし、`GITHUB_TOKEN` で作成されたイベントは[通常、別のワークフロー実行を開始しません][github-token-trigger]。これは tagpr に対して次の 2 箇所に影響します。

- tagpr が作成したタグは、`on.push.tags` で設定したワークフローを起動しない。
- tagpr が作成または更新したリリースプルリクエストを対象とする `pull_request` ワークフローは承認待ち状態で作成され、[書き込み権限を持つユーザーが承認する][bot-pr-approval]まで実行されない。

tagpr がタグを作成した後にリリースフローを自動で開始する方法は 2 つあります。

| 構成 | 利点 | トレードオフ |
| --- | --- | --- |
| tagpr ワークフロー内でリリースする | 追加の認証情報なしで `GITHUB_TOKEN` を使える | リリース PR ワークフローに承認が必要で、リリース処理が tagpr のワークフロー権限と環境を共有する |
| 別のリリースワークフローを起動する | リリースの責任を分離でき、タグとリリース PR のワークフローを自動実行できる | ワークフローを起動できるトークンが必要 |

## 同じワークフローでリリースする {#publish-in-the-same-workflow}

`tag` 出力は tagpr がタグを作成した場合にのみ空でなくなります。同じワークフロー内のリリース処理を実行する条件に使います。

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

この構成では GitHub App も personal access token も必要ありません。リリースロジックをスクリプトまたはローカルの composite action に置けば、tagpr とプロジェクト固有のリリース処理がワークフローを共有していても結合を抑えられます。完全な例は [Songmu/ecschedule's tagpr workflow][ecschedule-tagpr] を参照してください。

利用可能なその他の出力は次のとおりです。

- `pull_request`: リリースプルリクエストを説明する JSON。
- `base_tag`: 比較の基準に使った前回のタグ。最初のリリースでは空の値。

## 別のリリースワークフローを起動する {#trigger-a-separate-workflow}

リリースフローを `on.push.tags` で設定したワークフローに置くには、次のようにします。

```yaml
on:
  push:
    tags:
    - "v*"
```

タグでそのワークフローを起動できるよう、`GITHUB_TOKEN` 以外のトークンを tagpr に渡します。personal access token も使えますが、[`actions/create-github-app-token`][create-app-token] で作成する短命の GitHub App トークンを推奨します。

GitHub App は次の権限でリポジトリにインストールする必要があります。

- Contents: Read and write
- Pull requests: Read and write
- Issues: Read-only

App の作成、インストール、認証情報の保存については、`actions/create-github-app-token` のドキュメントを参照してください。設定後、トークンを生成し、checkout と tagpr の両方に使います。

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

このトークンによるタグの作成やリリースプルリクエストの更新では、`GITHUB_TOKEN` の制約を受けずに後続のワークフローを起動できます。

### リリースを復旧可能にする

どのワークフロー構成を選んでも、リリース処理を共通化しておき、明示的にタグを指定して個別に実行できるようにしておくと良いでしょう。それにより、失敗したビルド、公開、デプロイなどを、新しいリリースタグを作成せずに再実行できます。

たとえば、パッケージ化とアップロードのロジックをスクリプトまたはローカルの composite action に置いておく方法があります。これにより、tagpr ワークフローと復旧ワークフローの両方から同じリリース処理を呼び出せます。

```yaml
- name: Publish
  run: ./.github/scripts/publish "${{ inputs.tag }}"
```

## セキュリティ上の考慮事項

- 長期間有効な personal access token より、短命の GitHub App トークンを優先する。
- tagpr とリリース処理に必要な権限だけを付与する。
- checkout で認証情報がローカルの Git 設定に保持されないよう、`persist-credentials: false` を維持する。
- リポジトリのサプライチェーンポリシーに従って、サードパーティ Action を固定する。

Action の完全な出力リファレンスは、[README](../../README.md#outputs) を参照してください。

[bot-pr-approval]: https://github.blog/changelog/2026-06-11-bot-created-pull-requests-can-run-workflows-if-approved/
[create-app-token]: https://github.com/actions/create-github-app-token
[ecschedule-tagpr]: https://github.com/Songmu/ecschedule/blob/main/.github/workflows/tagpr.yaml
[github-token-trigger]: https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-when-your-workflow-runs/triggering-a-workflow
