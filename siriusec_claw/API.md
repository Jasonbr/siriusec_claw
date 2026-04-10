# SiriuSec Claw API 文档

## 基础信息

- **Base URL**: `http://localhost:3578`
- **WebSocket**: `ws://localhost:3578/ws`
- **Content-Type**: `application/json`

## 认证

目前版本无需认证。

---

## 系统接口

### 健康检查

```http
GET /health
```

**响应**:
```json
{
  "status": "healthy",
  "version": "0.0.1",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 系统状态

```http
GET /api/status
```

**响应**:
```json
{
  "version": "0.0.1",
  "uptime": "45h30m",
  "agents": 5,
  "sessions": 12,
  "channels": 3
}
```

---

## 聊天接口

### 发送消息 (REST)

```http
POST /api/chat
Content-Type: application/json
```

**请求体**:
```json
{
  "message": "检查 MySQL 状态",
  "session_id": "sess_xxx",
  "agent_id": "mysql-expert",
  "context": {}
}
```

**响应**:
```json
{
  "message_id": "msg_xxx",
  "content": "MySQL 状态正常...",
  "agent_id": "mysql-expert",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### WebSocket 连接

```
ws://localhost:3578/ws
```

**连接消息**:
```json
{
  "type": "connect",
  "session_id": "sess_xxx",
  "agent_id": "default"
}
```

**发送消息**:
```json
{
  "type": "message",
  "content": "检查服务器状态"
}
```

**接收消息**:
```json
{
  "type": "chunk",
  "content": "正在检查..."
}
```

---

## Agent 管理

### 获取 Agent 列表

```http
GET /api/agents
```

**响应**:
```json
{
  "agents": [
    {
      "id": "agent_xxx",
      "name": "MySQL 专家",
      "skill_ids": ["mysql"],
      "status": "active",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ]
}
```

### 创建 Agent

```http
POST /api/agents
Content-Type: application/json
```

**请求体**:
```json
{
  "name": "Docker 专家",
  "skill_ids": ["docker"],
  "model": "gpt-4",
  "config": {
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

### 获取 Agent 详情

```http
GET /api/agents/{agent_id}
```

### 更新 Agent

```http
PUT /api/agents/{agent_id}
Content-Type: application/json
```

### 删除 Agent

```http
DELETE /api/agents/{agent_id}
```

---

## 技能管理

### 获取技能列表

```http
GET /api/skills
```

**响应**:
```json
{
  "skills": [
    {
      "id": "mysql",
      "name": "MySQL 运维专家",
      "description": "专注于 MySQL 数据库运维管理",
      "tools": ["mysql_status", "mysql_slow_queries"],
      "version": "1.0.0"
    },
    {
      "id": "docker",
      "name": "Docker 运维专家",
      "description": "专注于 Docker 容器管理",
      "tools": ["docker_ps", "docker_stats"],
      "version": "1.0.0"
    }
  ]
}
```

### 获取技能详情

```http
GET /api/skills/{skill_id}
```

### 安装技能

```http
POST /api/skills/install
Content-Type: application/json
```

**请求体**:
```json
{
  "source": "github",
  "url": "https://github.com/xxx/skill-xxx"
}
```

---

## 会话管理

### 获取会话列表

```http
GET /api/sessions
```

**响应**:
```json
{
  "sessions": [
    {
      "id": "sess_xxx",
      "agent_id": "mysql-expert",
      "title": "MySQL 问题排查",
      "message_count": 15,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### 创建会话

```http
POST /api/sessions
Content-Type: application/json
```

**请求体**:
```json
{
  "agent_id": "docker-expert",
  "title": "Docker 容器排查"
}
```

### 获取会话详情

```http
GET /api/sessions/{session_id}
```

### 删除会话

```http
DELETE /api/sessions/{session_id}
```

### 获取会话消息

```http
GET /api/sessions/{session_id}/messages
```

---

## 渠道管理

### 获取渠道列表

```http
GET /api/channels
```

**响应**:
```json
{
  "channels": [
    {
      "id": "chan_xxx",
      "type": "dingtalk",
      "name": "钉钉通知",
      "status": "active",
      "config": {
        "webhook": "https://oapi.dingtalk.com/..."
      }
    }
  ]
}
```

### 创建渠道

```http
POST /api/channels
Content-Type: application/json
```

**请求体**:
```json
{
  "type": "dingtalk",
  "name": "运维告警",
  "config": {
    "webhook": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
    "secret": "xxx"
  }
}
```

### 更新渠道

```http
PUT /api/channels/{channel_id}
```

### 删除渠道

```http
DELETE /api/channels/{channel_id}
```

### 测试渠道

```http
POST /api/channels/{channel_id}/test
```

---

## MCP 管理

### 获取 MCP 服务器列表

```http
GET /api/mcp/servers
```

### 添加 MCP 服务器

```http
POST /api/mcp/servers
Content-Type: application/json
```

**请求体**:
```json
{
  "name": "文件系统服务",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
}
```

### 调用 MCP 工具

```http
POST /api/mcp/tools/call
Content-Type: application/json
```

**请求体**:
```json
{
  "server_name": "文件系统服务",
  "tool_name": "read_file",
  "arguments": {
    "path": "/tmp/test.txt"
  }
}
```

---

## 配置管理

### 获取配置

```http
GET /api/config
```

### 更新配置

```http
PUT /api/config
Content-Type: application/json
```

---

## 错误响应

### 错误格式

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "请求参数错误",
    "details": {
      "field": "agent_id",
      "issue": "required"
    }
  }
}
```

### 错误码

| 错误码 | HTTP 状态 | 说明 |
|--------|----------|------|
| `INVALID_REQUEST` | 400 | 请求参数错误 |
| `UNAUTHORIZED` | 401 | 未授权 |
| `FORBIDDEN` | 403 | 禁止访问 |
| `NOT_FOUND` | 404 | 资源不存在 |
| `INTERNAL_ERROR` | 500 | 服务器内部错误 |

---

## 分页

列表接口支持分页参数：

```http
GET /api/agents?page=1&page_size=20
```

**响应**:
```json
{
  "items": [...],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```
