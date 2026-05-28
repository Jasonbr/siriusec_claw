---
type: source
origin: user-note
date: 2026-05-27
tags:
  - llm-wiki/source
aliases: []
cssclasses: []
---


# 20-Decisions/2026-05-26-git-worktree-init.md

本文档记录了解决在空 Git 仓库上使用 git worktree 时出现 HEAD 无效错误的方案，包括先初始化 commit、使用 mkdir -p 以及统一 worktree 目录结构的决策。

## Entities

- [[opencode|OpenCode]]
- [[git|Git]]
- [[git-worktree-best-practices|Git Worktree Best Practices]]
- [[opencode-workflow-integration|Opencode Workflow Integration]]

## Concepts

- [[git-worktree|Git Worktree 初始化]]
- [[worktree|Worktree 目录结构规范]]
