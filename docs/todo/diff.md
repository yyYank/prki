# prki diff - 実装状況

## 概要

子PRマージ後の親PRの差分を確認するサブコマンド。

## 実装済み

なし。

## 未実装

### コマンド自体が存在しない

READMEの `prki merge` の説明内で言及されているが、`cmd/diff.go` が存在しない。

```bash
# マージ後の親PR差分確認
$ prki diff
```

**必要な作業:**
- `cmd/diff.go` を新規作成する
- `git diff main..HEAD` を使って親ブランチと main の差分を表示する
- 子PRがマージされた後の残差分をわかりやすく表示する
- `root.go` へのコマンド登録（`init()` 内で `rootCmd.AddCommand(diffCmd)`）
