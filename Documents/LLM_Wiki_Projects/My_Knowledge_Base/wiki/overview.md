# 知识库概览

## 当前状态

本知识库已初始化并处于活跃维护状态。目前专注于追踪 LLM Wiki 生态系统 —— 一套用于使用大语言模型管理个人知识库的工具和方法论。

## 结构

- **用途**: 个人知识管理，强调 LLM 增强的工作流
- **架构**: 多类型页面结构（实体、概念、来源、查询、综合、对比）
- **存储**: 本地文件系统，兼容 Obsidian
- **智能体规则**: 定义于 [[../AGENTS.md]] —— 指导 LLM 进行摄取、查询和检查操作

## 领域聚焦

### LLM Wiki 生态系统
本知识库目前涵盖两个主要来源：

1. **什么是 LLM Wiki** (2025-05-09)
   - 项目概览和核心特性介绍
   - 基础技术栈：Tauri、React、sigma.js、LanceDB

2. **GitHub - nashsu/llm_wiki** (2025-05-11)
   - 完整的项目 README，包含详细功能描述和架构设计
   - 新增了解的高级功能：
     - 4-Signal Knowledge Graph（四信号知识图谱关联模型）
     - Louvain Community Detection（Louvain 社区检测）
     - Vector Semantic Search（向量语义搜索）
     - Deep Research（深度研究）
     - Async Review System（异步审查系统）
     - SHA256 Incremental Cache（SHA256 增量缓存）
     - Purpose-Driven Wiki（目标驱动设计）
   - 多搜索 API 支持：Tavily、SerpApi、SearXNG
   - 多模态图像摄取和视觉 LLM 集成

### 追踪的关键技术
- **框架**: Tauri、React、Vite
- **图谱**: sigma.js、graphology、ForceAtlas2
- **搜索**: Tavily、SerpApi、SearXNG、LanceDB
- **工具**: Obsidian、Readability.js、Turndown.js
- **人物**: Andrej Karpathy、nashsu

## 工作流程

1. 将文档添加到 `raw/sources/`（通过网页剪藏、手动保存或复制粘贴）
2. 运行摄取以生成 wiki 页面（实体、概念、来源摘要）
3. 使用问题查询知识库
4. 探索生成的 wiki 页面及其关联
5. 通过检查流程审查和改进

## 统计

- 创建日期: 2025-04-28
- 首次摄取: 2025-05-09
- 最新摄取: 2025-05-11（通过 Playwright 网页剪藏）
- 来源数: 2
- Wiki 页面: 26（索引、日志、概览、10 个实体、11 个概念、2 个来源摘要）
- 活跃智能体: karpathy-llm-wiki 技能，通过 OpenCode + Playwright 进行网页剪藏

## 待解决问题

- 还应该向本知识库添加哪些领域？
- 随着更多来源被摄取，知识库将如何扩展？
- 是否应该安装 obsidian-llm-wiki 插件以进行替代查询？
- 我们能否将基于 Playwright 的网页剪藏集成到常规工作流程中？
