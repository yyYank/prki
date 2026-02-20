# prki (PR木)

**PR Tree** - A tool for splitting large Pull Requests into manageable child PRs.

巨大なPRを意味のあるかたまりで自動分割し、レビュー可能なサイズにするツール。

## Problem

AIコーディング時代、こんな問題ありませんか？

- Claude Code/Cursorで実装したら気づいたら2000行の巨大PR
- 「ちょっと修正」を頼んだら関連ファイル10個も変更された
- レビュー依頼したら「でかすぎる」と言われた
- スマホでレビューしたいけど画面に収まらない

**prki は「すでに大きくなったPR」を救済するツールです。**

## Solution

```bash
# 巨大PRを分析
$ prki analyze

🌳 PR木分析中...
親PR: feature/payment-system (1847行, 23ファイル)
  ├─ 子PR案1: Infrastructure & Config (200行, 5ファイル)
  ├─ 子PR案2: Core Business Logic (1200行, 12ファイル)  
  └─ 子PR案3: Tests & Documentation (447行, 6ファイル)

各子PRは独立してレビュー可能です。
分割しますか? [Y/n]

# 分割実行
$ prki split

✓ 子ブランチ review/config を作成
✓ 子ブランチ review/core を作成
✓ 子ブランチ review/tests を作成
✓ 子PR #101, #102, #103 を作成

次のステップ:
1. 子PR (#101-103) をレビュー依頼
2. 全て完了したら親PRをmainにマージ
```

## Features

### ✅ 事後的な分割
- すでに大きくなったPRを分析
- ローカルの未プッシュコミットも対応
- 「気づいたら大きくなってた」を救済

### ✅ 意味のあるグルーピング
- ディレクトリ構造で自動分類
- ファイルタイプ別（テスト/プロダクション/設定）
- セマンティック分析（リファクタ/機能/修正）

### ✅ 親子PR構造
- 親PR: 全体の変更（main ← feature）
- 子PR: レビュー用の部分的変更（feature ← review/xxx）
- 子PRは独立してレビュー可能
- 子PRマージ後、親PRで統合確認

### ✅ レビュー負荷軽減
- 小さい単位でレビュー可能
- スマホでも見やすいサイズ
- コンテキスト明確（「設定だけ」「テストだけ」）

## Installation

```bash
go install github.com/yyYank/prki@latest
```

## Quick Start

### Case 1: 既存の巨大PRを分割

```bash
# GitHubのPRを指定
$ prki analyze --pr 123

# または、ローカルブランチを指定  
$ prki analyze --branch feature/payment

# 分割実行
$ prki split --strategy semantic

# 子PRの状態確認
$ prki status
```

### Case 2: ローカルの大きいコミットを分割

```bash
# 現在のブランチを分析
$ prki analyze

# まだコミットしてない変更も分析可能
$ prki analyze --unstaged

# 分割してそれぞれブランチ作成
$ prki split
```

## Usage

### `prki analyze`

現在の変更を分析し、分割案を提示

```bash
# 基本
$ prki analyze

# GitHubのPR指定
$ prki analyze --pr 123

# ブランチ指定
$ prki analyze --branch feature/payment

# 分割戦略指定
$ prki analyze --strategy directory  # ディレクトリ単位
$ prki analyze --strategy filetype   # ファイルタイプ単位
$ prki analyze --strategy semantic   # 意味単位（デフォルト）

# 閾値カスタマイズ
$ prki analyze --threshold 500  # 500行超えたら分割提案
```

### `prki split`

分割を実行し、子ブランチ・子PRを作成

```bash
# 基本（対話式）
$ prki split

# 自動実行
$ prki split --auto

# 分割数指定
$ prki split --parts 3

# DraftでPR作成
$ prki split --draft

# レビュアー指定
$ prki split --reviewers alice,bob
```

### `prki status`

親子PRの状態を確認

```bash
$ prki status

親PR: #100 feature/payment-system
  ├─ 子PR #101: review/config [approved ✓]
  ├─ 子PR #102: review/core [changes requested]
  └─ 子PR #103: review/tests [pending review]

次のアクション:
- PR #102 の修正対応
```

## Workflow

### 典型的なワークフロー:

```bash
# 1. AIで実装（気づいたら大きくなってた）
$ cursor "決済機能を実装して"
# → 2000行の変更が...

# 2. 分析
$ prki analyze
# → 3つの子PRに分割可能

# 3. 分割実行
$ prki split
# → 子PR #101, #102, #103 作成

# 4. レビュー依頼（小さいので早い）
# → レビュアーがそれぞれレビュー

# 5. 親PRをレビュー（差分は統合部分のみ）
# → 最終確認

# 6. 親PRをmainにマージ
# → 完了！
```

## Configuration

`.prkirc` または `.prki.yaml` で設定可能:

```yaml
# 分割戦略
strategy: semantic  # semantic | directory | filetype

# 閾値
thresholds:
  files: 10        # ファイル数がこれを超えたら分割提案
  lines: 500       # 行数がこれを超えたら分割提案
  complexity: 100  # 複雑度がこれを超えたら分割提案

# グルーピングルール
grouping:
  - name: "Infrastructure & Config"
    patterns:
      - "*.config.{js,ts}"
      - "package.json"
      - "tsconfig.json"
      - ".github/**"
    order: 1

  - name: "Core Business Logic"
    patterns:
      - "src/**/*.{ts,tsx}"
    exclude:
      - "**/*.test.{ts,tsx}"
    order: 2

  - name: "Tests"
    patterns:
      - "**/*.test.{ts,tsx}"
      - "**/*.spec.{ts,tsx}"
    order: 3

  - name: "Documentation"
    patterns:
      - "*.md"
      - "docs/**"
    order: 4

# PRテンプレート
pr_template:
  child:
    title: "[Review] {group_name}"
    body: |
      This is a child PR for review purposes only.
      
      Parent PR: #{parent_pr_number}
      Group: {group_name}
      
      ## Changes
      {file_list}
      
      ## Context
      This PR is part of a larger feature. Please review this subset independently.
      Once approved, it will be merged into the parent branch.

# GitHub設定
github:
  create_draft: true       # 子PRをDraftで作成
  auto_assign_reviewers: true
  add_labels:
    - "review-split"
    - "child-pr"
```

## Examples

### Example 1: AIコーディング後の分割

```bash
# Claude Codeで実装
$ claude-code "ユーザー認証機能を実装"

# 差分確認
$ git diff --stat
 23 files changed, 1847 insertions(+)

# 分析
$ prki analyze
🌳 分割案:
  ├─ Config & Dependencies (package.json, tsconfig.json) - 3 files
  ├─ Auth Core (services/auth.ts, models/user.ts) - 8 files
  ├─ UI Components (components/Login.tsx, etc) - 7 files
  └─ Tests (*.test.ts) - 5 files

# 分割
$ prki split --auto

# 子PRレビュー
# ...レビュアーが各PRをレビュー...
```

### Example 2: 途中で大きくなった場合

```bash
# 開発中
$ git status
 modified: 15 files

# 分析（まだコミットしてない）
$ prki analyze --unstaged
⚠️  変更が大きいです (500行)

分割推奨:
  ├─ リファクタリング (8 files, 300 lines)
  └─ 新機能 (7 files, 200 lines)

# 分割してコミット
$ prki split --unstaged
✓ ブランチ refactor を作成してコミット
✓ ブランチ feature を作成してコミット

現在のブランチ: refactor
次: git checkout feature
```

## Why prki?

### vs 既存のStacked PR ツール（Graphite, SPR, stack-pr）

| | prki | Graphite/SPR/stack-pr |
|---|---|---|
| **ユースケース** | すでに大きいPRを分割 | 最初から小さく作る |
| **タイミング** | 事後的（治療） | 事前的（予防） |
| **粒度** | 意味のかたまり | コミット単位 |
| **構造** | 親子PR（水平分割） | スタックPR（垂直積み上げ） |
| **AIコーディング対応** | ◎ | △ |

### prki が必要な理由:

1. **AIが勝手に実装しすぎる時代**
   - 「ちょっと修正」→ 10ファイル変更
   - 計画的に小さく作るのが困難

2. **現実的な開発フロー**
   - 「気づいたら大きくなってた」を救済
   - 完璧な計画は無理

3. **レビュー負荷の現実**
   - スマホでレビューしたい
   - 巨大PRは誰も見たくない

## Roadmap

- [ ] v0.1: 基本的な分析・分割機能
- [ ] v0.2: GitHub連携（PR作成・更新）
- [ ] v0.3: 複雑度分析
- [ ] v0.4: AIによるセマンティック分析
- [ ] v0.5: GitLab, Bitbucket対応
- [ ] v1.0: 安定版

## Contributing

PRお待ちしています！特に：

- 分割戦略の改善
- 言語別の最適化
- CI/CD統合
- ドキュメント改善

## License

Apache 2

## Author

Created by someone tired of giant PRs in the AI coding era.

---

**prki** - Because AI makes your PRs too big, and reviewers need a break.
