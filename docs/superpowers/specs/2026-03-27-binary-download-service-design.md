# Binary Download Service Design

## Project Overview

- **Project Name**: Binary Download Service
- **Type**: Web Application (FastAPI + SQLite)
- **Core Functionality**: 上传和管理 node_exporter/node-push-exporter 二进制文件，提供下载服务
- **Target Users**: 运维人员，需要在多台服务器部署 exporter 的场景

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   Browser                        │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│           FastAPI Application                    │
│  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Static UI  │  │      REST API           │  │
│  │  (HTML/CSS) │  │  - GET /api/files       │  │
│  └─────────────┘  │  - POST /api/upload     │  │
│                   │  - DELETE /api/files/:id│  │
│                   │  - GET /download/:name  │  │
│                   └────────────┬────────────┘  │
│                                │               │
│                   ┌────────────▼────────────┐  │
│                   │      SQLite DB          │  │
│                   │  (files metadata)       │  │
│                   └─────────────────────────┘  │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│           Local File System                      │
│              uploads/                            │
│   node_exporter-1.8.1-linux-amd64.tar.gz        │
│   node_exporter-1.8.1-linux-arm64.tar.gz        │
│   node-push-exporter-v1.0.0-linux-amd64         │
└─────────────────────────────────────────────────┘
```

## Functionality Specification

### 1. 文件上传 (POST /api/upload)
- 接受 multipart/form-data 上传
- 解析文件名提取: program, version, os, arch
- 文件命名格式: `{program}-{version}-{os}-{arch}.tar.gz` 或 `{program}-{version}-{os}-{arch}`
- 存储到 `uploads/` 目录
- 记录元数据到 SQLite

### 2. 文件列表 (GET /api/files)
- 返回所有已上传文件
- 字段: id, filename, program, version, os, arch, file_size, uploaded_at
- 支持按 program 筛选 (query param: ?program=node_exporter)

### 3. 文件删除 (DELETE /api/files/{id})
- 根据 ID 删除数据库记录
- 同时删除文件系统中的实际文件

### 4. 文件下载 (GET /download/{filename})
- 从 uploads/ 目录提供文件下载
- 设置正确的 Content-Type 和 Content-Disposition

### 5. 前端页面 (GET /)
- 展示文件列表（按 program 分组）
- 上传表单（文件选择 + 自动解析显示）
- 删除按钮
- 复制下载链接按钮

## Data Model

### SQLite Table: files

```sql
CREATE TABLE files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL UNIQUE,
    program TEXT NOT NULL,           -- 'node_exporter' 或 'node-push-exporter'
    version TEXT NOT NULL,
    os TEXT NOT NULL,                -- 'linux', 'darwin', etc.
    arch TEXT NOT NULL,              -- 'amd64', 'arm64', etc.
    file_path TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_program ON files(program);
CREATE INDEX idx_program_version ON files(program, version);
```

## File Naming Convention

| 示例文件名 | program | version | os | arch |
|------------|---------|---------|-----|------|
| node_exporter-1.8.1-linux-amd64.tar.gz | node_exporter | 1.8.1 | linux | amd64 |
| node_exporter-1.8.1-linux-arm64.tar.gz | node_exporter | 1.8.1 | linux | arm64 |
| node-push-exporter-v1.0.0-linux-amd64 | node-push-exporter | v1.0.0 | linux | amd64 |

解析正则: `^(node_exporter|node-push-exporter)-(.+)-(linux|darwin)-(amd64|arm64)(\.tar\.gz)?$`

## API Specification

### GET /api/files
Response:
```json
{
  "files": [
    {
      "id": 1,
      "filename": "node_exporter-1.8.1-linux-amd64.tar.gz",
      "program": "node_exporter",
      "version": "1.8.1",
      "os": "linux",
      "arch": "amd64",
      "file_size": 12345678,
      "uploaded_at": "2026-03-27T10:00:00"
    }
  ]
}
```

### POST /api/upload
Request: multipart/form-data (file field)
Response:
```json
{
  "id": 1,
  "filename": "node_exporter-1.8.1-linux-amd64.tar.gz",
  "message": "文件上传成功"
}
```

### DELETE /api/files/{id}
Response:
```json
{
  "message": "文件删除成功"
}
```

## UI Design

- 简洁的单页面应用
- 顶部标题 + 两个 Program 分组 (node_exporter / node-push-exporter)
- 每个分组下: 上传按钮 + 文件列表
- 文件行显示: 版本、架构、大小、上传时间、下载链接、删除按钮
- 无需权限校验

## Acceptance Criteria

1. [ ] FastAPI 服务能正常启动
2. [ ] 可以通过 Web 界面上传文件
3. [ ] 上传后文件存储到 uploads/ 目录
4. [ ] 文件元数据保存到 SQLite
5. [ ] 文件列表正确显示所有上传的文件
6. [ ] 可以删除文件（同时删除 DB 记录和实际文件）
7. [ ] 可以通过 /download/{filename} 下载文件
8. [ ] 前端页面正常展示，可交互