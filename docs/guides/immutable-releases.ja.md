# Immutable Releases の活用と連携

tagpr はデフォルトでタグと公開済み GitHub Release を作成します。[immutable releases][github-immutable-releases] を有効にするリポジトリでは、公開前にビルドとアセットの添付を完了する必要があります。公開後は、関連付けられたタグを移動できず、リリースアセットを変更または削除できないためです。

GitHub は、draft release を作成し、すべてのアセットを添付し、完成した時点でのみ draft を公開することを推奨しています。immutable release を有効にする前に、その一連の処理をどのツールが所有するかを選んでください。

## リリースの所有者を選ぶ

tagpr は、後続のビルドまたは公開ワークフローと連携する 2 つのパターンをサポートします。

| 設定                      | 使う場合                         | リリースの所有者                                  |
| ----------------------- | ---------------------------- | ----------------------------------------- |
| `tagpr.release = draft` | tagpr にリリースタイトルとノートを生成させたい場合 | tagpr が draft を作成し、後続ワークフローがアセットを追加して公開する |
| `tagpr.release = false` | タグ作成時に厳密な境界を設けたい場合           | 後続ワークフローがリリースを作成し、内容を追加して公開する             |

後続ワークフローが immutable release にアセットをアップロードする場合、`tagpr.release = true` のままにしないでください。この設定ではリリースが直ちに公開されるため、後続ワークフローの実行時にはすでに immutable になっています。

どちらのパターンでも、下流の処理が明示的なタグを受け取り、安全に再実行できるようにしてください。そのタグ専用の draft を更新し、別のタグや 2 つ目のリリースを作成しないようにします。

## tagpr に draft release を準備させる

tagpr が生成したタイトルとリリースノートを保持するには、`release = draft` を設定します。

```ini
[tagpr]
    release = draft
```

tagpr がタグを作成した後、後続のステップでアセットをビルドし、既存の draft にアップロードして公開できます。

```yaml
- id: tagpr
  uses: Songmu/tagpr@v1
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

- name: Build release assets
  if: steps.tagpr.outputs.tag != ''
  run: ./scripts/build-release "${{ steps.tagpr.outputs.tag }}"

- name: Upload assets and publish the draft
  if: steps.tagpr.outputs.tag != ''
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAG: ${{ steps.tagpr.outputs.tag }}
  run: |
    gh release upload "$TAG" dist/project.tar.gz dist/checksums.txt --clobber
    gh release edit "$TAG" --draft=false
```

コマンドが draft を公開する前に、アップロードが成功しなければなりません。`GITHUB_TOKEN` を使う場合は、同じワークフロー内で公開を続けてください。そのトークンで作成されたタグは、通常、別のワークフローを起動しません。別のタグトリガーワークフローを使うには、ワークフローを起動できるトークンを tagpr に渡します。

tagpr は GitHub Release を作成する前にタグを push するため、draft が存在する前に別のワークフローが開始されることがあります。draft が利用可能になるまで検索を再試行するか、tagpr ジョブの完了後に公開ワークフローをディスパッチしてください。トークンを変更するとワークフローを起動できるようになりますが、それだけではこの競合を防げません。[タグ付けとリリース](tag-and-release.md#github_token-constraints) で、認証とワークフロー構成のトレードオフを説明しています。

`--clobber` は、同じタグからビルドを再現できるアセットにだけ使ってください。これにより、途中で完了した以前の試行がアップロードしたアセットを再試行時に置き換えられます。失敗またはキャンセルされたワークフローの後に未公開の draft が残っていないか監視し、明示的なタグを指定して処理を再実行してください。

### GoReleaser で draft を再利用する {#reuse-the-draft-with-goreleaser}

GoReleaser v2.5 以降では、tagpr が作成した draft に内容を追加できます。上記の tagpr 設定と次を組み合わせます。

```yaml
# .goreleaser.yml
release:
  use_existing_draft: true
```

`release = draft` がないと、tagpr は immutable release に対して早すぎるタイミングで公開します。`use_existing_draft: true` がないと、GoReleaser は tagpr の draft にアセットを追加せず、別のリリースを作成しようとすることがあります。完全な連携パターンは、[tagpr と GoReleaser の immutable release 連携][tagpr-goreleaser-article] で説明しています。

## 別のワークフローにリリースを作成させる

tagpr にリリースプルリクエストとタグだけを担当させる場合は、`release = false` を設定します。

```ini
[tagpr]
    release = false
```

その後、下流のワークフローがリリースノート、アセット、公開、復旧を担当します。たとえば、手動復旧用のタグ入力を受け取り、通常のタグトリガー実行では push されたタグにフォールバックします。

```yaml
- name: Create a draft and attach assets
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAG: ${{ inputs.tag || github.ref_name }}
  run: |
    if ! gh release view "$TAG" >/dev/null 2>&1; then
      gh release create "$TAG" --draft --generate-notes --verify-tag
    fi
    gh release upload "$TAG" \
      dist/project.tar.gz dist/checksums.txt \
      --clobber

- name: Publish the completed release
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAG: ${{ inputs.tag || github.ref_name }}
  run: gh release edit "$TAG" --draft=false
```

この分離は、既存のリリースツールまたはワークフローがリリースノートの生成をすでに管理している場合に便利です。tagpr が `GITHUB_TOKEN` で作成したタグは通常、タグトリガーワークフローを起動しないため、ワークフローが確実に起動するようにしてください。[別のリリースワークフローのガイダンス](tag-and-release.md#trigger-a-separate-workflow) では、GitHub App のインストールトークンを使う方法を説明しています。

この例では、`gh release create` を無条件に再実行するのではなく、既存の draft を検出して再利用します。アセット生成を決定的にし、必要なアップロードがすべて完了してから公開してください。

[github-immutable-releases]: https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases
[tagpr-goreleaser-article]: https://songmu.jp/riji/entry/2025-09-05-coordinate-tagpr-and-goreleaser-with-immutable-releases.html
