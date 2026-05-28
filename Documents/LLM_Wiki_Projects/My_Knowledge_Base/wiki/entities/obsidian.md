---
type: entity
entity-type: tool
name: Obsidian
aliases: []
tags:
  - llm-wiki/entity
  - llm-wiki/entity/tool
source-count: 5
date-updated: 2026-05-27
cssclasses: []
---


# Obsidian

## Facts

- 文档中提及.obsidian/目录用于存放该软件的配置文件。
- 生成的 Wiki 目录完全兼容该软件，可直接作为知识库打开。
- 作为目标知识库，通过 raw/sources/opencode-memory/ 路径接收 OpenCode 的同步文件。
- 作为知识库接收 OpenCode 同步的 Markdown 文件。
- 通过符号链接直接读取 OpenCode 的记忆文件，并作为 LLM Wiki 插件的运行环境。

## Connections

- [[llm-wiki]] *(uses)*
- [[llm-wiki]] *(related-to)*
- [[obsidian-web-clipper]] *(uses)*
- [[llm-wiki]] *(uses)*
- [[opencode]] *(uses)*
- [[sync-memory-to-obsidiansh]] *(applies-to)*
- [[]] *(applies-to)*

## Sources

- [[schema.md]]
- [[raw/sources/sample_article.md]]
- [[raw/sources/opencode-memory/test-auto-sync-2026-05-19.md]]
- [[raw/sources/opencode-memory/README.md]]
- [[raw/sources/opencode-memory/AUTO-SYNC-SETUP-COMPLETE.md]]
