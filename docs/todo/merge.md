# prki merge - 実装状況

## 概要

承認された子PRを親ブランチにマージするサブコマンド。

## 実装済み

- [x] コマンド自体の登録
- [x] `--pr` フラグの定義（マージ対象のPR番号指定）

## 未実装

### マージ処理本体 (`cmd/merge.go:27`)

コマンドはスタブのみで、実際のマージ処理が全く実装されていない。

```go
// TODO: fetch approved child PRs via GitHub API and merge
fmt.Println("\nNot yet implemented. For now, run manually:")
fmt.Println("  gh pr merge <child-pr-number> --merge --delete-branch")
```

**必要な作業:**
- `--pr` 未指定時: `gh pr list` で承認済み子PRを自動検索する
- `--pr` 指定時: 指定されたPR番号のみを対象にする
- `gh pr merge <number> --merge --delete-branch` を実行してマージする
- マージ後に親ブランチの差分確認を促すメッセージを表示する
