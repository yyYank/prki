# prki status - 実装状況

## 概要

親子PRの状態を確認するサブコマンド。

## 実装済み

- [x] 現在のブランチ名の表示
- [x] ローカルの変更ファイル数・行数の表示
- [x] 子PRの状態取得 (`cmd/status.go`)
  - `gh pr list --base <branch>` で現在ブランチを親とする子PRを取得
  - 各子PRのレビュー状態（approved / changes requested / pending review）を表示
  - 次のアクション（修正対応すべきPR、マージ可能なPR）を表示

## 実装の詳細

### 追加した型・関数

- `ChildPR` 構造体: GitHub APIから取得した子PRの情報 (number, title, reviewDecision)
- `fetchChildPRs(branch string)`: `gh` CLIを使って子PRを取得
- `reviewLabel(decision string) string`: GitHub のレビュー状態を表示用文字列に変換
- `nextActions(prs []ChildPR)`: PRをtoFix/toMergeに分類

### 出力例

```
Current branch: feature/payment-system

Parent PR: feature/payment-system
  ├─ Child PR #101: review/config [approved ✓]
  ├─ Child PR #102: review/core [changes requested]
  └─ Child PR #103: review/tests [pending review]

Next actions:
  • Fix changes requested: review/core
  • Ready to merge: review/config

Current changes:
  3 files, 150 lines
```

### gh CLI が存在しない場合

```
Parent PR: feature/payment-system
  (Could not fetch child PR status: gh CLI not found (https://cli.github.com))
```

## ユニットテスト

`cmd/status_test.go` にて以下をカバー:

- `TestReviewLabel`: 各レビュー状態文字列 (大文字・小文字) のラベル変換
- `TestNextActions`: 混在パターンでのtoFix/toMerge分類
- `TestNextActions_AllApproved`: 全承認時
- `TestNextActions_AllChangesRequested`: 全変更要求時
- `TestNextActions_AllPending`: 全保留時
- `TestNextActions_Empty`: 空リスト
- `TestNextActions_CaseInsensitive`: 小文字の reviewDecision も正しく処理
