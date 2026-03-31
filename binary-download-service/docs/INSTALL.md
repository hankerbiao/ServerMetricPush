# Binary Download Service 安装部署文档

## 概述

Binary Download Service 是一个用于上传、存储和分发二进制文件的 FastAPI 服务。主要功能包括：

- 二进制文件管理（node_exporter、node-push-exporter 等）
- 节点注册与心跳监控
- Web 界面管理控制台

## 环境要求

| 要求 | 版本 | 说明 |
|------|------|------|
| Python | >= 3.9 | 推荐 3.11+ |
| 操作系统 | Linux / macOS | Windows 下也可运行 |
| 磁盘空间 | 至少 500MB | 用于存储二进制文件 |

## 安装步骤

### 1. 克隆项目

```bash
git clone <repository-url>
cd binary-download-service
```

### 2. 创建虚拟环境（推荐）

```bash
# 使用 venv
python -m venv venv
source venv/bin/activate  # Linux/macOS
# 或
venv\\Scripts\\activate  # Windows

# 或使用 uv
uv venv
source .venv/bin/activate  # Linux/macOS
```

### 3. 安装依赖

```bash
pip install -r requirements.txt
```

### 4. 初始化目录

首次运行时会自动创建以下目录：

- `uploads/` - 存储上传的二进制文件
- `files.db` - SQLite 数据库文件

## 运行服务

### 方式一：使用启动脚本

```bash
./run.sh
```

### 方式二：直接运行

```bash
uvicorn main:app --host 0.0.0.0 --port 8080
```

### 方式三：开发模式（热重载）

```bash
uvicorn main:app --reload --host 0.0.0.0 --port 8080
```

## 验证安装

服务启动后，在浏览器中访问：

| 地址 | 说明 |
|------|------|
| http://localhost:8080 | 文件管理控制台 |
| http://localhost:8080/agents | 节点状态监控 |

API 健康检查：

```bash
curl http://localhost:8080/api/files
```

响应示例：
```json
{"files":[]}
```

## 配置说明

### 端口配置

默认端口为 8080。如需修改，编辑 `run.sh` 或启动命令：

```bash
uvicorn main:app --host 0.0.0.0 --port 9000
```

### 心跳配置

在 `main.py` 中可调整以下参数：

```python
HEARTBEAT_INTERVAL_SECONDS = 30    # 节点心跳间隔（秒）
OFFLINE_TIMEOUT_SECONDS = 90       # 节点离线超时（秒）
RECENT_EVENTS_LIMIT = 20           # 保留最近事件数量
```

### 上传目录

默认上传目录为项目根目录下的 `uploads/`。如需修改，编辑 `main.py`：

```python
UPLOAD_DIR = os.path.join(os.path.dirname(__file__), "your/custom/path")
```

## 目录结构

```
binary-download-service/
├── main.py              # FastAPI 应用主入口
├── database.py          # SQLAlchemy 数据库模型
├── schemas.py           # Pydantic 数据模型
├── requirements.txt     # Python 依赖
├── run.sh              # 服务启动脚本
├── CLAUDE.md           # 项目说明文档
├── docs/               # 文档目录
│   ├── INSTALL.md      # 本文档
│   ├── USER_GUIDE.md   # 用户使用指南
│   └── API.md          # API 接口手册
├── uploads/            # 上传文件存储目录（自动创建）
└── files.db            # SQLite 数据库（自动创建）
```

## 部署建议

### 生产环境

1. 使用反向代理（如 Nginx）
2. 配置 HTTPS
3. 设置防火墙规则
4. 定期备份 `files.db` 和 `uploads/` 目录

### Nginx 配置示例

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Docker 部署（可选）

如需 Docker 部署，可创建以下 Dockerfile：

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .

EXPOSE 8080
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8080"]
```

## 常见问题

### Q: 上传文件失败

检查 `uploads/` 目录是否有写入权限：

```bash
chmod 755 uploads/
```

### Q: 数据库连接错误

确保 `files.db` 所在目录有写入权限，或删除后让服务自动重建。

### Q: 节点心跳超时

检查节点与服务器的 网络连通性，确保防火墙允许对应端口通信。

## 卸载

```bash
# 停止服务
# 删除项目目录
rm -rf /path/to/binary-download-service

# 如需保留数据，请备份 uploads/ 和 files.db
```