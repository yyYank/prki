# prki check - 実装TODO

## 概要

変更サイズが閾値を超えていたら非ゼロ終了するサブコマンド。
CI・コミットフックに組み込んでPRが大きくなる前に止める。

## ユースケース

```bash
# pre-push hook
prki check --threshold 500 || exit 1

# GitHub Actions
- run: prki check --threshold 300
```

## 必要な作業

- `check` サブコマンドを追加する
- `analyze` のロジック（`getChangedFiles` + 行数集計）を再利用する
- 閾値超過時に非ゼロ終了 + エラーメッセージを出力する
- `--threshold` フラグを持つ（デフォルト500）
- `--branch` フラグも持つ（`analyze` と同様）

## 出力イメージ

```
# 正常時
✓ Change size is within threshold (320 lines < 500)

# 閾値超過時 (exit code 1)
✗ Change size exceeds threshold (1847 lines > 500)
  Run `prki analyze` to see split proposal.
```
