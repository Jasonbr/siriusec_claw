# SiriuSec Claw

SiriuSec Claw 是一个开源的企业级智能体平台，专为运维场景打造。基于 Go 语言开发，采用单二进制架构，内嵌前端，支持多 Agent 协作、故障自愈、多渠道集成等特性。

## 特性

- **多 Agent 协作** - 支持多个智能体协同工作，自动任务分配
- **运维技能库** - 内置 MySQL、Docker、K8s、Linux、Prometheus、Nginx 等运维专家技能
- **故障自愈** - 自动巡检、告警检测、自动修复
- **多渠道集成** - 支持钉钉、企业微信、Telegram、飞书、Slack 等
- **MCP 支持** - 完整的 Model Context Protocol 协议支持
- **RAG 知识库** - 向量存储、相似度搜索、智能问答
- **WebSocket 实时通信** - 支持流式响应和实时消息推送
- **单二进制部署** - 单文件部署，内嵌前端，零依赖

## 快速开始

### 使用 Docker

```bash
docker run -p 3578:3578 jasonbr/siriusec_claw:latest
```

### 本地运行

```bash
# 克隆仓库
git clone https://github.com/Jasonbr/siriusec_claw.git
cd siriusec_claw

# 启动服务
make run-dev
```

### 访问

打开浏览器访问 `http://localhost:3578`

## 构建

### 构建前端

```bash
cd ui && npm install && npm run build
```

### 构建后端

```bash
cd src && go build -o ../bin/siriusec_claw ./cmd/siriusec_claw
```

### 构建 Docker 镜像

```bash
docker build -t siriusec_claw:latest .
```

## 目录结构

```
.
├── cmd/              # 主程序入口
├── embed/            # 嵌入资源（前端、技能）
│   ├── frontend/     # 前端构建产物
│   └── skills/       # 技能定义
├── pkg/              # 核心包
│   ├── agent/        # Agent 管理
│   ├── channels/     # 多渠道集成
│   ├── config/       # 配置管理
│   ├── gateway/      # 网关服务
│   ├── mcp/          # MCP 协议实现
│   └── ...
└── ui/               # 前端源码
```

## API 文档

### REST API

#### 健康检查

```http
GET /health
```

响应：
```json
{
  "status": "healthy",
  "version": "0.0.1",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### 聊天接口

```http
POST /api/chat
Content-Type: application/json

{
  "message": "检查 MySQL 状态",
  "session_id": "sess_xxx",
  "agent_id": "mysql-expert"
}
```

#### WebSocket 连接

```
ws://localhost:3578/ws
```

### 技能管理 API

#### 获取技能列表

```http
GET /api/skills
```

#### 获取技能详情

```http
GET /api/skills/{skill_id}
```

### Agent 管理 API

#### 获取 Agent 列表

```http
GET /api/agents
```

#### 创建 Agent

```http
POST /api/agents
Content-Type: application/json

{
  "name": "MySQL 专家",
  "skill_ids": ["mysql"],
  "model": "gpt-4"
}
```

### 渠道管理 API

#### 获取渠道列表

```http
GET /api/channels
```

#### 配置渠道

```http
POST /api/channels
Content-Type: application/json

{
  "type": "dingtalk",
  "name": "钉钉通知",
  "config": {
    "webhook": "https://oapi.dingtalk.com/robot/send?access_token=xxx"
  }
}
```

## 配置

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `CLAW_PORT` | 服务端口 | `3578` |
| `CLAW_DEBUG` | 调试模式 | `false` |
| `CLAW_LOG_LEVEL` | 日志级别 | `info` |
| `OPENAI_API_KEY` | OpenAI API 密钥 | - |
| `OPENAI_API_BASE` | OpenAI API 地址 | `https://api.openai.com/v1` |

### 配置文件

配置文件位于 `~/.config/siriusec_claw/config.yaml`：

```yaml
server:
  port: 3578
  debug: false

agents:
  - name: "MySQL 专家"
    skill: "mysql"
    model: "gpt-4"

channels:
  - type: "dingtalk"
    webhook: "https://..."
```

## 技能开发

### 技能定义文件

在 `embed/skills/{skill_name}/SKILL.md` 创建技能定义：

```markdown
# 技能名称

## 描述
技能描述...

## 能力
- 能力1
- 能力2

## 工具
### tool_name
工具描述...
```json
{
  "name": "tool_name",
  "description": "...",
  "parameters": {...}
}
```

## 工作流
步骤说明...

## 示例对话
用户: ...
助手: ...
```

## 故障自愈

### 配置自动巡检

```yaml
inspection:
  enabled: true
  interval: 5m
  tasks:
    - name: "磁盘空间检查"
      command: "df -h"
    - name: "内存检查"
      command: "free -h"
```

### 配置告警规则

```yaml
alerts:
  - name: "CPU 过高"
    condition: "cpu_usage > 85"
    severity: "warning"
    auto_heal: true
    heal_action: "notify"
```

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/xxx`)
3. 提交更改 (`git commit -am 'Add xxx'`)
4. 推送分支 (`git push origin feature/xxx`)
5. 创建 Pull Request

## 许可证

MIT License

## 联系方式

- GitHub: https://github.com/Jasonbr/siriusec_claw
- Email: jasonbr@example.com
