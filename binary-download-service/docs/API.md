# Binary Download Service API 接口手册

## 概述

本文档详细介绍 Binary Download Service 提供的所有 REST API 接口。

**Base URL:** `http://localhost:8080`

**注意:** 所有 API 返回 JSON 格式数据。

---

## 目录

- [文件管理 API](#文件管理-api)
  - [获取文件列表](#获取文件列表)
  - [上传文件](#上传文件)
  - [删除文件](#删除文件)
  - [下载文件](#下载文件)
- [节点管理 API](#节点管理-api)
  - [注册节点](#注册节点)
  - [节点心跳](#节点心跳)
  - [获取节点列表](#获取节点列表)
  - [获取节点详情](#获取节点详情)

---

## 文件管理 API

### 获取文件列表

获取所有已上传的文件列表。

**Endpoint:** `GET /api/files`

**Query Parameters:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| program | string | 否 | 筛选程序类型：`node_exporter`, `node-push-exporter`, `install-script` |

**Response:**

```json
{
  "files": [
    {
      "id": 1,
      "filename": "node_exporter-1.5.0-linux-amd64.tar.gz",
      "program": "node_exporter",
      "version": "1.5.0",
      "os": "linux",
      "arch": "amd64",
      "file_size": 12345678,
      "uploaded_at": "2024-01-15T10:30:00"
    }
  ]
}
```

**FileResponse 字段说明:**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | integer | 文件唯一 ID |
| filename | string | 文件名 |
| program | string | 程序类型 |
| version | string | 版本号 |
| os | string | 操作系统 |
| arch | string | 架构 |
| file_size | integer | 文件大小（字节） |
| uploaded_at | string | 上传时间 |

**示例:**

```bash
# 获取所有文件
curl http://localhost:8080/api/files

# 筛选 node_exporter
curl "http://localhost:8080/api/files?program=node_exporter"
```

---

### 上传文件

上传二进制文件到服务器。

**Endpoint:** `POST /api/upload`

**Content-Type:** `multipart/form-data`

**Request Body:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 要上传的文件 |

**文件名格式要求:**

| 程序 | 格式 | 示例 |
|------|------|------|
| node_exporter | `{version}-{os}-{arch}.tar.gz` | node_exporter-1.5.0-linux-amd64.tar.gz |
| node-push-exporter | `{version}-{os}-{arch}.tar.gz` | node-push-exporter-1.0.0-darwin-arm64.tar.gz |
| 安装脚本 | `install.sh` 或 `uninstall.sh` | install.sh |

**Response (成功):**

```json
{
  "id": 1,
  "filename": "node_exporter-1.5.0-linux-amd64.tar.gz",
  "message": "文件上传成功"
}
```

**Response (失败):**

```json
{
  "detail": "错误信息"
}
```

**示例:**

```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@node_exporter-1.5.0-linux-amd64.tar.gz"
```

---

### 删除文件

删除指定文件。

**Endpoint:** `DELETE /api/files/{file_id}`

**Path Parameters:**

| 参数 | 类型 | 说明 |
|------|------|------|
| file_id | integer | 文件 ID |

**Response (成功):**

```json
{
  "message": "文件删除成功"
}
```

**Response (失败):**

```json
{
  "detail": "文件不存在"
}
```

**示例:**

```bash
curl -X DELETE http://localhost:8080/api/files/1
```

---

### 下载文件

下载指定文件。

**Endpoint:** `GET /download/{filename}`

**Path Parameters:**

| 参数 | 类型 | 说明 |
|------|------|------|
| filename | string | 文件名 |

**Response:**

- 成功: 返回文件内容，Content-Type 为 `application/octet-stream`
- 失败: 返回 404 错误

**示例:**

```bash
# 直接下载
curl -O http://localhost:8080/download/node_exporter-1.5.0-linux-amd64.tar.gz

# 指定保存文件名
curl -o my-file.tar.gz http://localhost:8080/download/node_exporter-1.5.0-linux-amd64.tar.gz
```

---

## 节点管理 API

### 注册节点

注册或更新 node-push-exporter 节点。

**Endpoint:** `POST /api/agents/register`

**Request Body:**

```json
{
  "agent_id": "unique-agent-id",
  "hostname": "server-01",
  "version": "1.0.0",
  "os": "linux",
  "arch": "amd64",
  "ip": "192.168.1.5",
  "pushgateway_url": "http://prometheus:9091",
  "push_interval_seconds": 30,
  "node_exporter_port": 9100,
  "node_exporter_metrics_url": "http://localhost:9100/metrics",
  "started_at": "2024-01-15T10:00:00"
}
```

**AgentRegisterRequest 字段说明:**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_id | string | 是 | 节点唯一标识符 |
| hostname | string | 是 | 主机名 |
| version | string | 是 | 程序版本 |
| os | string | 是 | 操作系统 (`linux`, `darwin`) |
| arch | string | 是 | 架构 (`amd64`, `arm64`) |
| ip | string | 是 | 节点 IP 地址 |
| pushgateway_url | string | 是 | Pushgateway 地址 |
| push_interval_seconds | integer | 是 | 推送间隔（秒） |
| node_exporter_port | integer | 是 | node_exporter 端口 |
| node_exporter_metrics_url | string | 是 | node_exporter 指标 URL |
| started_at | string | 是 | 启动时间 (ISO 8601 格式) |

**Response:**

```json
{
  "heartbeat_interval_seconds": 30,
  "offline_timeout_seconds": 90
}
```

**AgentRegisterResponse 字段说明:**

| 字段 | 类型 | 说明 |
|------|------|------|
| heartbeat_interval_seconds | integer | 心跳间隔（秒） |
| offline_timeout_seconds | integer | 离线超时（秒） |

**示例:**

```bash
curl -X POST http://localhost:8080/api/agents/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "server-01",
    "hostname": "server-01",
    "version": "1.0.0",
    "os": "linux",
    "arch": "amd64",
    "ip": "192.168.1.5",
    "pushgateway_url": "http://prometheus:9091",
    "push_interval_seconds": 30,
    "node_exporter_port": 9100,
    "node_exporter_metrics_url": "http://localhost:9100/metrics",
    "started_at": "2024-01-15T10:00:00"
  }'
```

---

### 节点心跳

节点定期发送心跳更新状态。

**Endpoint:** `POST /api/agents/heartbeat`

**Request Body:**

```json
{
  "agent_id": "unique-agent-id",
  "status": "online",
  "last_error": null,
  "last_push_at": "2024-01-15T10:30:00",
  "last_push_success_at": "2024-01-15T10:30:00",
  "last_push_error_at": null,
  "push_fail_count": 0,
  "node_exporter_up": true
}
```

**AgentHeartbeatRequest 字段说明:**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_id | string | 是 | 节点唯一标识符 |
| status | string | 是 | 状态 (`online`, `degraded`, `offline`) |
| last_error | string | 否 | 最近错误信息 |
| last_push_at | string | 否 | 最后推送时间 |
| last_push_success_at | string | 否 | 最后成功推送时间 |
| last_push_error_at | string | 否 | 最后推送失败时间 |
| push_fail_count | integer | 是 | 连续推送失败次数 |
| node_exporter_up | boolean | 是 | node_exporter 是否运行中 |

**Response (成功):**

```json
{
  "message": "心跳更新成功"
}
```

**Response (失败):**

```json
{
  "detail": "节点未注册"
}
```

**示例:**

```bash
curl -X POST http://localhost:8080/api/agents/heartbeat \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "server-01",
    "status": "online",
    "last_error": null,
    "last_push_at": "2024-01-15T10:30:00",
    "last_push_success_at": "2024-01-15T10:30:00",
    "last_push_error_at": null,
    "push_fail_count": 0,
    "node_exporter_up": true
  }'
```

---

### 获取节点列表

获取所有已注册的节点列表。

**Endpoint:** `GET /api/agents`

**Response:**

```json
{
  "agents": [
    {
      "agent_id": "server-01",
      "hostname": "server-01",
      "version": "1.0.0",
      "os": "linux",
      "arch": "amd64",
      "ip": "192.168.1.5",
      "status": "online",
      "online": true,
      "last_error": null,
      "pushgateway_url": "http://prometheus:9091",
      "push_interval_seconds": 30,
      "node_exporter_port": 9100,
      "node_exporter_metrics_url": "http://localhost:9100/metrics",
      "node_exporter_up": true,
      "push_fail_count": 0,
      "started_at": "2024-01-15T10:00:00",
      "last_seen_at": "2024-01-15T10:30:00",
      "last_push_at": "2024-01-15T10:30:00",
      "last_push_success_at": "2024-01-15T10:30:00",
      "last_push_error_at": null,
      "registered_at": "2024-01-15T10:00:00",
      "updated_at": "2024-01-15T10:30:00"
    }
  ]
}
```

**AgentResponse 字段说明:**

| 字段 | 类型 | 说明 |
|------|------|------|
| agent_id | string | 节点唯一标识符 |
| hostname | string | 主机名 |
| version | string | 程序版本 |
| os | string | 操作系统 |
| arch | string | 架构 |
| ip | string | IP 地址 |
| status | string | 有效状态 (`online`, `degraded`, `offline`) |
| online | boolean | 是否在线（基于心跳超时判断） |
| last_error | string | 最近错误信息 |
| pushgateway_url | string | Pushgateway 地址 |
| push_interval_seconds | integer | 推送间隔 |
| node_exporter_port | integer | node_exporter 端口 |
| node_exporter_metrics_url | string | node_exporter 指标 URL |
| node_exporter_up | boolean | node_exporter 是否运行 |
| push_fail_count | integer | 连续推送失败次数 |
| started_at | string | 启动时间 |
| last_seen_at | string | 最后收到心跳时间 |
| last_push_at | string | 最后推送时间 |
| last_push_success_at | string | 最后成功推送时间 |
| last_push_error_at | string | 最后推送失败时间 |
| registered_at | string | 注册时间 |
| updated_at | string | 更新时间 |

**示例:**

```bash
curl http://localhost:8080/api/agents
```

---

### 获取节点详情

获取指定节点的详细信息及最近事件。

**Endpoint:** `GET /api/agents/{agent_id}`

**Path Parameters:**

| 参数 | 类型 | 说明 |
|------|------|------|
| agent_id | string | 节点 ID |

**Response:**

```json
{
  "agent": { ... },
  "events": [
    {
      "id": 1,
      "agent_id": "server-01",
      "event_type": "registered",
      "message": "server-01 已完成注册",
      "created_at": "2024-01-15T10:00:00"
    }
  ]
}
```

**AgentDetailResponse 字段说明:**

| 字段 | 类型 | 说明 |
|------|------|------|
| agent | AgentResponse | 节点信息 |
| events | array[AgentEventResponse] | 最近事件列表 |

**AgentEventResponse 字段说明:**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | integer | 事件 ID |
| agent_id | string | 节点 ID |
| event_type | string | 事件类型 (`registered`, `reregistered`, `status_changed`, `error`) |
| message | string | 事件消息 |
| created_at | string | 事件发生时间 |

**Response (失败):**

```json
{
  "detail": "节点不存在"
}
```

**示例:**

```bash
curl http://localhost:8080/api/agents/server-01
```

---

## 错误响应

所有 API 在发生错误时返回标准错误响应：

```json
{
  "detail": "错误描述信息"
}
```

### 常见错误码

| 状态码 | 说明 |
|--------|------|
| 404 | 资源不存在 |
| 422 | 请求参数验证失败 |
| 500 | 服务器内部错误 |

---

## OpenAPI / Swagger

服务内置了 OpenAPI 文档，可通过以下地址访问：

- Swagger UI: http://localhost:8080/docs
- ReDoc: http://localhost:8080/redoc

---

## 相关文档

- [安装部署文档](INSTALL.md)
- [用户使用指南](USER_GUIDE.md)