# タグ付けとリリース

tagpr では、バージョンタグの作成をリリースフローの起点とします。タグ付け後のリリースフローでは、成果物のビルドやパッケージ化、パッケージや GitHub Release アセットの公開、アプリケーションのデプロイなどを実行できます。

タグは、リリース対象のソースとバージョンを一意に示します。後続のステップでは tagpr の `tag` 出力を利用でき、タグの push を契機として別のリリースワークフローを起動することもできます。

> [!NOTE]
> リポジトリで immutable release を有効にし、後続の処理で GitHub Release にアセットを追加する場合は、すべてのアセットを添付してから公開する必要があります。`tagpr.release = draft` と `tagpr.release = false` の連携パターンは、[Immutable Releases の活用と連携](immutable-releases.md) を参照してください。

## 署名付きタグ

tagpr は Git 標準の `tag.gpgSign` 設定を尊重します。`true` の場合は
`Release <tag>` というメッセージを持つ署名付き annotated tag を作成します。
`false` または未設定の場合は、従来どおり lightweight tag を作成します。

署名方式、鍵、agent、証明書、Git identity は、tagpr の実行前に設定してください。
tagpr は署名鍵の import や管理を行いません。たとえば GPG、SSH、X.509 のいずれかを
設定した後、次のように署名付きタグを有効にします。

```yaml
- name: Enable signed tags
  run: git config --global tag.gpgSign true
```

署名が有効でも Git が署名を作成できない場合、tagpr はタグを push せずに失敗します。

GitHub Actions で keyless 署名を利用する場合は、
[Chainguard の `setup-gitsign` Action][setup-gitsign]を使って workflow の OIDC identity
を Git の署名に利用できます。

```yaml
permissions:
  contents: write
  pull-requests: write
  issues: read
  id-token: write

steps:
- uses: actions/checkout@v6
  with:
    persist-credentials: false
- uses: chainguard-dev/actions/setup-gitsign@805da2efdffdc42b8afd8880e575a48b471ef544 # v1.6.37
- uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

`setup-gitsign` 自身が `tag.gpgSign` を有効にするため、追加の Git 設定は不要です。
長期間有効な署名鍵ではなく、短命な Sigstore 証明書を利用します。現時点では GitHub は
gitsign による commit や tag の署名を `Verified` と表示しません。検証が必要な場合は
[`gitsign verify-tag`][gitsign-verification]を利用してください。

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

`tag` 出力はタグの作成または復旧が成功した場合に空でなくなります。同じワークフロー内のリリース処理を実行する条件に使います。
成功した再試行でも既存の一致するタグが返り、後続の処理が再実行されることがあるため、これらの処理も再試行に対応させてください。

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

## タグ作成前にテストと承認を行う {#test-and-approve-before-tagging}

> [!NOTE]
> **Experimental:** `plan` / `apply` モードと plan JSON の形式は実験的な機能であり、
> 今後のリリースで変更される可能性があります。

デフォルトでは、tagpr はマージ済みのリリースプルリクエストを検出すると、すぐにタグを作成します。
マージ後の正確なコミットを追加でテストする場合や、protected environment の承認を待ってからタグを
作成する場合は、`plan` モードと `apply` モードを利用します。

`plan` モードは、通常どおりリリースプルリクエストを作成・更新します。マージ済みの
リリースプルリクエストを検出すると、タグを作成せず、リリース計画を単一の `plan` JSON として
出力します。プルリクエストの作成・更新だけを行った場合、`plan` 出力は空です。
JSON には次の情報が含まれます。

- `tag`: 作成予定のバージョンタグ。
- `commit_sha`: テスト・タグ付けする、マージされたリリースコミットの正確な SHA。
- `base_tag`: 直前のリリースタグ。最初のリリースでは空文字。
- `pull_request_number`: マージされたリリースプルリクエストの番号。
- `release_notes_boundary_sha`: リリースノート生成に使う終点の境界。

最終 job では、JSON を変更せず `apply` モードの `plan` 入力に渡します。tagpr はリポジトリの状態と
plan を再検証してからタグを作成します。承認待ちの間に対象コミットがリリースブランチの先頭ではなくなっても、
設定されたリリースブランチの ancestor であればタグを作成できます。

plan 適用後のリリース処理には、従来の `tag`、`pull_request`、`base_tag` 出力を利用します。
適用前のテストでは、`fromJSON(...).commit_sha` でテスト対象のコミットを checkout します。

```yaml
name: tagpr
on:
  push:
    branches:
    - main
  workflow_dispatch:

# 承認待ちの run を維持し、後続の main push は 1 つの pending run にまとめる。
concurrency:
  group: tagpr-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: false

permissions:
  contents: write
  pull-requests: write
  issues: read

jobs:
  plan:
    runs-on: ubuntu-latest
    outputs:
      plan: ${{ steps.plan.outputs.plan }}
    steps:
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
    - id: plan
      uses: Songmu/tagpr@v1
      with:
        mode: plan
      env:
        GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}

  test-release:
    needs: plan
    if: needs.plan.outputs.plan != ''
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
    - uses: actions/checkout@v6
      with:
        ref: ${{ fromJSON(needs.plan.outputs.plan).commit_sha }}
        persist-credentials: false
    - run: go test ./...

  apply:
    needs:
    - plan
    - test-release
    if: needs.plan.outputs.plan != '' && needs.test-release.result == 'success'
    runs-on: ubuntu-latest
    environment: release
    permissions:
      contents: write
      pull-requests: read
    steps:
    # この token は protected environment の承認後に作成される。
    - name: Generate token
      id: app-token
      uses: actions/create-github-app-token@v3
      with:
        client-id: ${{ secrets.CLIENT_ID }}
        private-key: ${{ secrets.PRIVATE_KEY }}
        permission-contents: write
        permission-pull-requests: read
    - uses: actions/checkout@v6
      with:
        ref: ${{ fromJSON(needs.plan.outputs.plan).commit_sha }}
        token: ${{ steps.app-token.outputs.token }}
        persist-credentials: false
    - id: apply
      uses: Songmu/tagpr@v1
      with:
        mode: apply
        plan: ${{ needs.plan.outputs.plan }}
      env:
        GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
```

リポジトリの **Settings → Environments** で `release` environment を作成し、required reviewers を
設定します。GitHub は `apply` を開始する前に承認を待つため、environment secrets と最終 job の
短命 token は承認前には使われません。30 日間承認されなかった deployment は
[自動的に失敗します][deployment-approval]。

この構成では workflow-level concurrency が必須です。plan が承認を待っている間に、別の
リリースブランチ push から同じ未リリース範囲に対する `plan` を開始してはいけません。
`cancel-in-progress: false` にすると、承認待ちの run は継続し、後続の run は最大 1 つの pending run に
まとめられます。monorepo で複数の tagpr 設定を使う場合は、設定ファイルのパスなど、リリース系列を
識別できる安定した値を concurrency group に含めます。

plan 適用は再実行できます。タグの push が成功した後に GitHub Release の作成だけ失敗した場合は、
同じ plan で最終 job を再実行すると、既存タグが `commit_sha` を指していることを確認したうえで
Release 作成を再試行します。同じ名前のタグが別のコミットを指している場合は失敗します。

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
[deployment-approval]: https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/control-deployments
[ecschedule-tagpr]: https://github.com/Songmu/ecschedule/blob/main/.github/workflows/tagpr.yaml
[gitsign-verification]: https://github.com/sigstore/gitsign#verifying-commits
[github-token-trigger]: https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-when-your-workflow-runs/triggering-a-workflow
[setup-gitsign]: https://github.com/chainguard-dev/actions/tree/main/setup-gitsign
