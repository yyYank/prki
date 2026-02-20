# prki analyze - 実装状況

## 概要

現在の変更を分析し、分割案を提示するサブコマンド。

## 実装済み

- [x] 基本的な分析 (`prki analyze`)
- [x] ブランチ指定 (`--branch`)
- [x] 閾値カスタマイズ (`--threshold`)
- [x] 分割戦略指定 (`--strategy`: `semantic` / `directory` / `filetype`)

## 未実装

### `--pr` フラグ (`cmd/analyze.go`)

フラグ自体は定義されているが、実際には使われていない。

```go
// analyzePR 変数は定義されているが getChangedFiles() に渡されていない
analyzeCmd.Flags().IntVar(&analyzePR, "pr", 0, "GitHub PR number to analyze")
```

**必要な作業:**
- `gh pr diff <pr-number> --name-only` 等を使って指定PRの変更ファイルを取得する処理を実装する
- `getChangedFiles()` を PR番号に対応させるか、別関数を用意する

### `--unstaged` フラグ

フラグ自体が未定義。

**必要な作業:**
- `--unstaged` フラグを追加する
- `git diff --numstat`（ステージ前の変更）を取得する処理を実装する
