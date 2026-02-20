# Sample Code - prki Core Implementation

## analyzer.py - 分析エンジン

```python
"""
Analyzer: 変更を分析し、分割案を生成
"""

from dataclasses import dataclass
from typing import List, Dict, Optional
from pathlib import Path
import subprocess
import re


@dataclass
class FileChange:
    """単一ファイルの変更情報"""
    path: str
    lines_added: int
    lines_deleted: int
    complexity: int = 0
    
    @property
    def total_lines_changed(self) -> int:
        return self.lines_added + self.lines_deleted


@dataclass
class FileGroup:
    """分割されたファイルグループ"""
    name: str
    files: List[FileChange]
    order: int  # マージ順序
    
    @property
    def total_lines(self) -> int:
        return sum(f.total_lines_changed for f in self.files)
    
    @property
    def complexity(self) -> int:
        return sum(f.complexity for f in self.files)
    
    @property
    def risk_level(self) -> str:
        if self.complexity < 50:
            return "low"
        elif self.complexity < 100:
            return "medium"
        else:
            return "high"


@dataclass
class SplitProposal:
    """分割提案"""
    groups: List[FileGroup]
    total_files: int
    total_lines: int
    
    def display(self):
        """分割案を表示"""
        print("\n🌳 PR木分析中...\n")
        print(f"現在の変更: {self.total_files}ファイル, {self.total_lines}行\n")
        print("分割案:")
        
        for i, group in enumerate(self.groups, 1):
            emoji = {"low": "✓", "medium": "⚠️", "high": "🔴"}[group.risk_level]
            print(f"  {i}. {group.name} {emoji}")
            print(f"     - {len(group.files)}ファイル, {group.total_lines}行")
            print(f"     - 複雑度: {group.risk_level}")
            print(f"     - ファイル:")
            for file in group.files[:3]:  # 最初の3つだけ表示
                print(f"       • {file.path}")
            if len(group.files) > 3:
                print(f"       ... and {len(group.files) - 3} more")
            print()


class Analyzer:
    """変更を分析し分割案を生成"""
    
    def __init__(self, base_branch: str = "main"):
        self.base_branch = base_branch
    
    def analyze(self, branch: Optional[str] = None) -> SplitProposal:
        """ブランチまたは現在の変更を分析"""
        # 変更ファイルを取得
        files = self._get_changed_files(branch)
        
        # グルーピング
        groups = self._group_files_semantic(files)
        
        # 複雑度計算
        self._calculate_complexity(files)
        
        # 提案作成
        total_files = len(files)
        total_lines = sum(f.total_lines_changed for f in files)
        
        return SplitProposal(groups, total_files, total_lines)
    
    def _get_changed_files(self, branch: Optional[str] = None) -> List[FileChange]:
        """変更ファイル一覧を取得"""
        if branch:
            # ブランチ指定
            diff_cmd = f"git diff {self.base_branch}..{branch} --numstat"
        else:
            # 現在の変更
            diff_cmd = f"git diff {self.base_branch}..HEAD --numstat"
        
        result = subprocess.run(
            diff_cmd.split(),
            capture_output=True,
            text=True
        )
        
        files = []
        for line in result.stdout.strip().split('\n'):
            if not line:
                continue
            
            parts = line.split('\t')
            if len(parts) != 3:
                continue
            
            added, deleted, path = parts
            
            # バイナリファイルは除外
            if added == '-' or deleted == '-':
                continue
            
            files.append(FileChange(
                path=path,
                lines_added=int(added),
                lines_deleted=int(deleted)
            ))
        
        return files
    
    def _group_files_semantic(self, files: List[FileChange]) -> List[FileGroup]:
        """ファイルを意味のあるかたまりでグルーピング"""
        groups = {
            "Infrastructure & Config": [],
            "Core Business Logic": [],
            "UI & Components": [],
            "Tests": [],
            "Documentation": []
        }
        
        for file in files:
            path_lower = file.path.lower()
            
            # 分類
            if self._is_test_file(path_lower):
                groups["Tests"].append(file)
            elif self._is_config_file(path_lower):
                groups["Infrastructure & Config"].append(file)
            elif self._is_doc_file(path_lower):
                groups["Documentation"].append(file)
            elif self._is_ui_file(path_lower):
                groups["UI & Components"].append(file)
            else:
                groups["Core Business Logic"].append(file)
        
        # 空のグループを削除し、FileGroupに変換
        result = []
        order = 1
        for name, file_list in groups.items():
            if file_list:
                result.append(FileGroup(
                    name=name,
                    files=file_list,
                    order=order
                ))
                order += 1
        
        return result
    
    def _is_test_file(self, path: str) -> bool:
        """テストファイルかどうか"""
        patterns = [
            r'\.test\.',
            r'\.spec\.',
            r'/tests?/',
            r'/__tests__/',
        ]
        return any(re.search(pattern, path) for pattern in patterns)
    
    def _is_config_file(self, path: str) -> bool:
        """設定ファイルかどうか"""
        patterns = [
            r'package\.json',
            r'tsconfig\.json',
            r'\.config\.(js|ts)',
            r'\.yml$',
            r'\.yaml$',
            r'/\.github/',
            r'Dockerfile',
        ]
        return any(re.search(pattern, path) for pattern in patterns)
    
    def _is_doc_file(self, path: str) -> bool:
        """ドキュメントファイルかどうか"""
        patterns = [
            r'\.md$',
            r'/docs?/',
            r'README',
        ]
        return any(re.search(pattern, path) for pattern in patterns)
    
    def _is_ui_file(self, path: str) -> bool:
        """UIファイルかどうか"""
        patterns = [
            r'/components?/',
            r'/pages?/',
            r'/views?/',
            r'\.tsx$',
            r'\.jsx$',
            r'\.vue$',
        ]
        return any(re.search(pattern, path) for pattern in patterns)
    
    def _calculate_complexity(self, files: List[FileChange]):
        """複雑度を計算（簡易版）"""
        for file in files:
            # 行数ベースの簡易計算
            # 本格実装ではAST解析が必要
            base = file.total_lines_changed // 10
            
            # ファイルタイプで調整
            ext = Path(file.path).suffix
            if ext in ['.ts', '.tsx', '.js', '.jsx']:
                multiplier = 1.2
            elif ext in ['.py']:
                multiplier = 0.9
            elif ext in ['.go']:
                multiplier = 0.8
            else:
                multiplier = 1.0
            
            file.complexity = int(base * multiplier)


# 使用例
if __name__ == "__main__":
    analyzer = Analyzer()
    proposal = analyzer.analyze()
    proposal.display()
    
    # 出力例:
    # 🌳 PR木分析中...
    # 
    # 現在の変更: 23ファイル, 1847行
    # 
    # 分割案:
    #   1. Infrastructure & Config ✓
    #      - 5ファイル, 200行
    #      - 複雑度: low
    #      - ファイル:
    #        • package.json
    #        • tsconfig.json
    #        • .github/workflows/ci.yml
    #        ... and 2 more
    # 
    #   2. Core Business Logic 🔴
    #      - 12ファイル, 1200行
    #      - 複雑度: high
    #      - ファイル:
    #        • src/services/payment.ts
    #        • src/models/transaction.ts
    #        • src/controllers/checkout.ts
    #        ... and 9 more
```

## splitter.py - 分割実行

```python
"""
Splitter: 分割案を実際のブランチ・PRに変換
"""

import subprocess
from typing import List, Optional
from dataclasses import dataclass


@dataclass
class ChildBranch:
    """子ブランチ情報"""
    name: str           # "review/config"
    group_name: str     # "Infrastructure & Config"
    files: List[str]    # ファイルパス一覧
    parent_branch: str  # "feature/payment"


@dataclass
class ChildPR:
    """子PR情報"""
    number: Optional[int]  # PR番号（作成後に設定）
    branch: ChildBranch
    url: Optional[str] = None


class Splitter:
    """分割を実行しブランチ・PRを作成"""
    
    def __init__(self, parent_branch: str):
        self.parent_branch = parent_branch
    
    def split(self, proposal) -> List[ChildPR]:
        """分割を実行"""
        child_prs = []
        
        for group in proposal.groups:
            # 子ブランチ作成
            branch = self._create_child_branch(group)
            
            # コミット作成
            self._commit_files(branch, group)
            
            # PR作成（GitHub CLI使用）
            pr = self._create_pull_request(branch)
            
            child_prs.append(pr)
        
        return child_prs
    
    def _create_child_branch(self, group) -> ChildBranch:
        """子ブランチを作成"""
        # ブランチ名を生成
        branch_name = self._generate_branch_name(group.name)
        
        # 親ブランチから分岐
        subprocess.run([
            "git", "checkout", "-b", branch_name, self.parent_branch
        ])
        
        # ファイルパス一覧
        file_paths = [f.path for f in group.files]
        
        return ChildBranch(
            name=branch_name,
            group_name=group.name,
            files=file_paths,
            parent_branch=self.parent_branch
        )
    
    def _generate_branch_name(self, group_name: str) -> str:
        """グループ名からブランチ名を生成"""
        # "Infrastructure & Config" -> "review/config"
        name = group_name.lower()
        name = name.split()[-1]  # 最後の単語を取得
        return f"review/{name}"
    
    def _commit_files(self, branch: ChildBranch, group):
        """該当ファイルのみをコミット"""
        # 全ファイルをリセット
        subprocess.run(["git", "reset", "HEAD"])
        
        # 該当ファイルのみをステージング
        for file in branch.files:
            subprocess.run(["git", "add", file])
        
        # コミットメッセージを入力
        print(f"\n{branch.name} のコミットメッセージを入力:")
        commit_msg = input("> ")
        
        # コミット
        subprocess.run(["git", "commit", "-m", commit_msg])
        
        # プッシュ
        subprocess.run(["git", "push", "-u", "origin", branch.name])
    
    def _create_pull_request(self, branch: ChildBranch) -> ChildPR:
        """GitHub PRを作成"""
        # GitHub CLI (gh) を使用
        result = subprocess.run([
            "gh", "pr", "create",
            "--base", self.parent_branch,
            "--head", branch.name,
            "--title", f"[Review] {branch.group_name}",
            "--body", self._generate_pr_body(branch),
            "--draft"
        ], capture_output=True, text=True)
        
        # PR URLを抽出
        pr_url = result.stdout.strip()
        
        # PR番号を抽出（URLから）
        pr_number = int(pr_url.split('/')[-1]) if pr_url else None
        
        return ChildPR(
            number=pr_number,
            branch=branch,
            url=pr_url
        )
    
    def _generate_pr_body(self, branch: ChildBranch) -> str:
        """PR本文を生成"""
        files_list = "\n".join(f"- {f}" for f in branch.files)
        
        return f"""
## Review Purpose

This is a child PR for review purposes only.

**Parent Branch:** `{self.parent_branch}`  
**Group:** {branch.group_name}

## Files in This PR

{files_list}

## Context

This PR is part of a larger feature split for easier review.
Once approved, it will be merged back into the parent branch.

Please review this subset independently.
"""


# 使用例
if __name__ == "__main__":
    from analyzer import Analyzer
    
    # 1. 分析
    analyzer = Analyzer()
    proposal = analyzer.analyze()
    proposal.display()
    
    # 2. 確認
    print("\n分割を実行しますか? [Y/n]")
    if input("> ").lower() != 'n':
        # 3. 分割実行
        splitter = Splitter(parent_branch="feature/payment")
        child_prs = splitter.split(proposal)
        
        # 4. 結果表示
        print("\n✓ 分割完了！\n")
        for pr in child_prs:
            print(f"✓ 子PR #{pr.number}: {pr.branch.group_name}")
            print(f"  {pr.url}")
```

## cli.py - コマンドラインインターフェース

```python
"""
CLI: コマンドラインインターフェース
"""

import click
from analyzer import Analyzer
from splitter import Splitter


@click.group()
def cli():
    """prki - PR Tree (PR木) - Split large PRs into manageable pieces"""
    pass


@cli.command()
@click.option('--branch', default=None, help='Branch to analyze')
@click.option('--pr', type=int, default=None, help='GitHub PR number')
@click.option('--threshold', type=int, default=500, help='Line threshold')
@click.option('--strategy', 
              type=click.Choice(['semantic', 'directory', 'filetype']),
              default='semantic',
              help='Grouping strategy')
def analyze(branch, pr, threshold, strategy):
    """Analyze changes and propose splits"""
    
    analyzer = Analyzer()
    
    if pr:
        # GitHub PRから分析（未実装）
        click.echo(f"Analyzing PR #{pr}...")
        # TODO: GitHub APIで差分取得
        return
    
    # ローカルブランチを分析
    proposal = analyzer.analyze(branch)
    
    # 閾値チェック
    if proposal.total_lines < threshold:
        click.echo(f"✓ 変更量は問題ありません ({proposal.total_lines}行)")
        return
    
    # 提案表示
    proposal.display()
    
    click.echo(f"\n推奨: {len(proposal.groups)}個の子PRに分割")
    click.echo(f"理由: レビュー負荷を軽減")


@cli.command()
@click.option('--auto', is_flag=True, help='Skip confirmation')
@click.option('--draft', is_flag=True, default=True, help='Create as draft PR')
@click.option('--reviewers', default=None, help='Comma-separated reviewer list')
def split(auto, draft, reviewers):
    """Execute the split and create child PRs"""
    
    # 1. 分析
    analyzer = Analyzer()
    proposal = analyzer.analyze()
    proposal.display()
    
    # 2. 確認
    if not auto:
        click.echo("\n分割を実行しますか? [Y/n]")
        if input("> ").lower() == 'n':
            click.echo("キャンセルしました")
            return
    
    # 3. 親ブランチを取得
    import subprocess
    result = subprocess.run(
        ["git", "rev-parse", "--abbrev-ref", "HEAD"],
        capture_output=True,
        text=True
    )
    parent_branch = result.stdout.strip()
    
    # 4. 分割実行
    splitter = Splitter(parent_branch)
    child_prs = splitter.split(proposal)
    
    # 5. 結果表示
    click.echo("\n✓ 分割完了！\n")
    for pr in child_prs:
        click.echo(f"✓ 子PR #{pr.number}: {pr.branch.group_name}")
        click.echo(f"  {pr.url}\n")
    
    click.echo("次のステップ:")
    click.echo("1. 各子PRをレビュー依頼")
    click.echo("2. 承認後: prki merge")


@cli.command()
def status():
    """Show status of parent and child PRs"""
    
    # TODO: GitHub APIで子PRの状態を取得
    click.echo("親PR: #100 feature/payment-system")
    click.echo("  ├─ 子PR #101: review/config [approved ✓]")
    click.echo("  ├─ 子PR #102: review/core [changes requested]")
    click.echo("  └─ 子PR #103: review/tests [pending review]")
    click.echo("\n次のアクション:")
    click.echo("- PR #102 の修正対応")
    click.echo("- PR #101, #103 承認後: prki merge")


@cli.command()
@click.option('--pr', multiple=True, type=int, help='Specific child PR numbers to merge')
def merge(pr):
    """Merge approved child PRs into parent branch"""
    
    if pr:
        click.echo(f"指定された子PR {pr} をマージします")
    else:
        click.echo("承認された全ての子PRをマージします")
    
    # TODO: 実装
    click.echo("\n✓ マージ完了")
    click.echo("親PRの差分を確認してください: git diff main")


if __name__ == "__main__":
    cli()
```

## 使用例

```bash
# 1. インストール
$ pip install -e .

# 2. 分析
$ prki analyze

🌳 PR木分析中...

現在の変更: 23ファイル, 1847行

分割案:
  1. Infrastructure & Config ✓
     - 5ファイル, 200行
     - 複雑度: low
     ...

# 3. 分割
$ prki split

分割を実行しますか? [Y/n] y

review/config のコミットメッセージを入力:
> Add infrastructure and config files

review/core のコミットメッセージを入力:
> Implement payment core logic

review/tests のコミットメッセージを入力:
> Add tests and documentation

✓ 分割完了！

✓ 子PR #101: Infrastructure & Config
  https://github.com/org/repo/pull/101

✓ 子PR #102: Core Business Logic
  https://github.com/org/repo/pull/102

✓ 子PR #103: Tests & Documentation
  https://github.com/org/repo/pull/103

# 4. 状態確認
$ prki status

親PR: #100 feature/payment-system
  ├─ 子PR #101: review/config [approved ✓]
  ├─ 子PR #102: review/core [approved ✓]
  └─ 子PR #103: review/tests [approved ✓]

# 5. マージ
$ prki merge

✓ マージ完了
親PRの差分を確認してください: git diff main
```

