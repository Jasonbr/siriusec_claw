# 活动日志

## 2025-04-28
- **创建**: 知识库已初始化
- **目的**: 个人知识管理系统
- **架构**: 定义了页面类型和结构

## [2025-05-09] 摄取 | 什么是 LLM Wiki

- **摘要**: 摄取了 raw/sources/ 目录下的 sample_article.md。提取了与 LLM Wiki 项目和方法论相关的关键实体和概念。
- **处理的来源**: [[raw/sources/sample_article.md]]
- **创建的来源页面**: [[wiki/sources/sample-article.md]]
- **创建的实体**: Andrej Karpathy、nashsu、Tauri、Obsidian、Tavily、LanceDB、sigma.js、graphology
- **创建的概念**: LLM Wiki、知识图谱、两步思维链录入、Chrome 网页剪藏、社区检测
- **更新的文件**: [[wiki/index.md]]、[[wiki/overview.md]]
- **方法**: 通过 OpenCode 使用 karpathy-llm-wiki 技能

## [2025-05-11] 摄取 | GitHub - nashsu/llm_wiki

- **摘要**: 使用 Playwright 浏览器自动化网页剪藏了 nashsu/llm_wiki 的 GitHub README。提取了详细的功能描述、架构设计和实现细节。为新实体和高级概念创建了全面的 wiki 页面。
- **处理的来源**: [[raw/sources/20250511-011617-github-nashsu-llm-wiki.md]]
- **创建的来源页面**: [[wiki/sources/20250511-011617-github-nashsu-llm-wiki.md]]
- **创建的实体**: SerpApi、SearXNG
- **创建的概念**: 四信号知识图谱、向量语义搜索、深度研究、异步审查系统、SHA256 增量缓存、目标驱动设计
- **更新的文件**: [[wiki/index.md]]、[[wiki/overview.md]]
- **方法**: Playwright 浏览器自动化 + 通过 OpenCode 使用 karpathy-llm-wiki 技能

## 模板
```
## [YYYY-MM-DD] 操作 | 标题

- 变更摘要
- 涉及的页面: [页面](relative/path.md)、[页面](relative/path.md)
```

## [2026-05-11] 摄取 | 批量摄取 (1 个文件)

- 通过 uv run ingest 自动摄取了 1 个来源
- 涉及的页面: [[wiki/sources/introduction-to-obsidian-web-clipper-obsidian-help.md]]
- **注意**: 完整的实体/概念提取需要 LLM 智能体审查

## [2026-05-11] 摄取 | 批量摄取 (1 个文件)

- 通过 uv run ingest 自动摄取了 1 个来源
- 涉及的页面: [[wiki/sources/introduction-to-obsidian-web-clipper-obsidian-help.md]]
- **注意**: 完整的实体/概念提取需要 LLM 智能体审查

## [2026-05-11] 摄取 | 批量摄取 (1 个文件)

- 通过 uv run ingest 自动摄取了 1 个来源
- 涉及的页面: introduction-to-obsidian-web-clipper-obsidian-help.md
- **注意**: 完整的实体/概念提取需要 LLM 智能体审查

## [2026-05-11] 检查 | 自动健康检查

- 检查了 27 个页面
- 发现 0 个死链接、0 个孤立页面
- 报告: .llm-wiki/lint-report.md


## [2026-05-11] 摄取 | 批量摄取 (1 个文件)

- 通过 uv run ingest 自动摄取了 1 个来源
- 涉及的页面: test-automation.md
- **注意**: 完整的实体/概念提取需要 LLM 智能体审查

## [2026-05-18] 检查 | 自动健康检查

- 检查了 63 个页面
- 发现 159 个死链接、19 个孤立页面
- 报告: .llm-wiki/lint-report.md


## [2026-05-18] 检查 | 自动健康检查

- 检查了 63 个页面
- 发现 101 个死链接、19 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]


## [2026-05-18] 检查 | 自动健康检查

- 检查了 64 个页面
- 发现 102 个死链接、20 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]


## [2026-05-19] 检查 | 自动健康检查

- 检查了 78 个页面
- 发现 134 个死链接、16 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]


## [2026-05-20] 检查 | 自动健康检查

- 检查了 78 个页面
- 发现 135 个死链接、16 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]


## [2026-05-21] 检查 | 自动健康检查

- 检查了 78 个页面
- 发现 136 个死链接、16 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]


## [2026-05-22] 检查 | 自动健康检查

- 检查了 78 个页面
- 发现 137 个死链接、16 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]


## [2026-05-25] 检查 | 自动健康检查

- 检查了 76 个页面
- 发现 138 个死链接、15 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]


## [2026-05-26] 检查 | 自动健康检查

- 检查了 76 个页面
- 发现 139 个死链接、15 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]


## [2026-05-27] 检查 | 自动健康检查

- 检查了 79 个页面
- 发现 144 个死链接、15 个孤立页面
- 报告: [[../../.llm-wiki/lint-report.md]]

