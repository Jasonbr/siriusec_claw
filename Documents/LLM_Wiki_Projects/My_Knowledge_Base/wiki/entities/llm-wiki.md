---
type: entity
entity-type: project
name: LLM Wiki
aliases:
  - llm_wiki
tags:
  - llm-wiki/entity
  - llm-wiki/entity/project
source-count: 7
date-updated: 2026-05-27
cssclasses: []
---


# LLM Wiki

## Facts

- 基于卡帕西理念的开源项目，将文档自动转化为结构化、相互关联的个人知识库。
- 一个跨平台桌面应用程序，可将文档自动转化为结构化、相互链接的个人知识库。
- 采用三层架构：原始源文件、大语言模型生成的维基、模式与配置。
- 生成的维基目录完全兼容 Obsidian，可直接作为 Obsidian 库使用。
- 作为知识图谱可视化的目标数据源，其内链关系被提取用于构建图数据。
- 维护一个持久的 Markdown 维基以跨来源积累理解
- 包含 raw/（只读源材料）和 wiki/（维护的知识层）目录结构
- 提供 OnSaveWatcher 监控机制，负责检测新文件并自动提取概念、实体与关系。
- 自动提取同步到 Obsidian 的 Markdown 文件并生成知识图谱。
- 提供 OnSaveWatcher 与夜间调度器，负责从同步的 Markdown 文件中自动提取知识。
- 其配置文件 data.json 中启用了夜间提取并设定执行时间为每日 17:00。

## Connections

- [[]] *(influences)*
- [[obsidian]] *(related-to)*
- [[github]] *(related-to)*
- [[tauri]] *(uses)*
- [[react]] *(uses)*
- [[sigmajs]] *(uses)*
- [[forceatlas2]] *(uses)*
- [[tavily-api]] *(uses)*
- [[mozilla-readabilityjs]] *(uses)*
- [[turndownjs]] *(uses)*
- [[lancedb]] *(uses)*
- [[andrej-karpathy]] *(extends)*
- [[nashsu]] *(created-by)*
- [[rag]] *(contrasts-with)*
- [[obsidian]] *(uses)*
- [[serpapi]] *(uses)*
- [[searxng]] *(uses)*
- [[graphology]] *(uses)*
- [[rust]] *(uses)*
- [[chrome-web-clipper]] *(uses)*
- [[schemamd]] *(uses)*
- [[purposemd]] *(uses)*
- [[louvain]] *(uses)*
- [[]] *(uses)*
- [[onsavewatcher]] *(uses)*
- [[myknowledgebase]] *(applies-to)*
- [[]] *(part-of)*
- [[chrome]] *(part-of)*
- [[]] *(applies-to)*
- [[llm]] *(applies-to)*
- [[opencode]] *(uses)*
- [[obsidian]] *(uses)*
- [[qwen36-plus]] *(applies-to)*

## Sources

- [[raw/sources/sample_article.md]]
- [[raw/sources/20250511-011617-github-nashsu-llm-wiki.md]]
- [[docs/plans/2026-05-11-knowledge-graph-viz.md]]
- [[AGENTS.md]]
- [[raw/sources/opencode-memory/test-auto-sync-2026-05-19.md]]
- [[raw/sources/opencode-memory/README.md]]
- [[raw/sources/opencode-memory/AUTO-SYNC-SETUP-COMPLETE.md]]
