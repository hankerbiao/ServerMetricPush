# node-push-exporter API 参考手册

本文档提供 node-push-exporter 的配置格式和可选控制面 API 的技术参考。

> **注意**: node-push-exporter 本身不提供 HTTP API 服务，其核心功能是推送指标到 Pushgateway。本文档描述配置文件格式和控制面 API。

## 配置文件格式

### 文件格式

配置文件采用 **key=value** 格式，每行一个配置项。

```bash
# 注释行以 # 开头
pushgateway.url=http://localhost:9091
pushgateway.job=node
```

### 完整配置示例

```bash
# ====================
# Pushgateway 配置
# ====================
pushgateway.url=http://10.17.154.252:9091
pushgateway.job=node
pushgateway.instance=
pushgateway.interval=60
pushgateway.timeout=10

# ====================
# node_exporter 配置
# ====================
node_exporter.path=node_exporter
node_exporter.port=9100
node_exporter.metrics_url=http://localhost:9100/metrics

# ====================
# 控制面配置 (可选)
# ====================
control_plane.url=http://10.17.154.252:8888
control_plane.heartbeat_interval=30
```

### 配置项索引

| 配置项 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| pushgateway.url | 是 | - | Pushgateway 服务地址 |
| pushgateway.job | 是 | - | 推送到 Pushgateway 的 job 标签 |
| pushgateway.instance | 否 | 本机 IP | 推送到 Pushgateway 的 instance 标签 |
| pushgateway.interval | 是 | - | 推送间隔（秒） |
| pushgateway.timeout | 否 | 10 | HTTP 请求超时（秒） |
| node_exporter.path | 是 | - | node_exporter 可执行文件路径 |
| node_exporter.port | 否 | 9100 | node_exporter 监听端口 |
| node_exporter.metrics_url | 是 | - | 抓取指标的完整 URL |
| control_plane.url | 否* | - | 控制面服务地址 |
| control_plane.heartbeat_interval | 否* | - | 心跳发送间隔（秒） |

*control_plane.url 和 control_plane.heartbeat_interval 必须同时配置或同时不配置

## 控制面 API

当启用控制面功能时，node-push-exporter 会向控制面服务注册并发送心跳。

### 注册接口

**端点**: `POST /api/agents/register`

**请求体**:

```json
{
  "agent_id": "a1b2c3d4e5f6...",
  "hostname": "web-server-01",
  "version": "v1.0.0",
  "os": "linux",
  "arch": "amd64",
  "ip": "192.168.1.100",
  "pushgateway_url": "http://10.17.154.252:9091",
  "push_interval_seconds": 60,
  "node_exporter_port": 9100,
  "node_exporter_metrics_url": "http://localhost:9100/metrics",
  "started_at": "2024-01-15T10:00:00Z"
}
```

**响应体**:

```json
{
  "heartbeat_interval_seconds": 30,
  "offline_timeout_seconds": 120
}
```

### 心跳接口

**端点**: `POST /api/agents/heartbeat`

**请求体**:

```json
{
  "agent_id": "a1b2c3d4e5f6...",
  "status": "online",
  "last_error": "",
  "last_push_at": "2024-01-15T10:01:00Z",
  "last_push_success_at": "2024-01-15T10:01:00Z",
  "last_push_error_at": null,
  "push_fail_count": 0,
  "node_exporter_up": true
}
```

**状态值说明**:

| status | 说明 |
|--------|------|
| online | 正常运行 |
| degraded | 运行异常（node_exporter 宕机或推送失败） |

### 错误响应

**状态码**: 4xx / 5xx

```json
{
  "error": "error message"
}
```

## Prometheus Pushgateway 集成

### 推送格式

node-push-exporter 使用 Prometheus text format 0.0.4 推送指标。

### URL 构造规则

```
{pushgateway.url}/metrics/job/{job}[/instance/{instance}]
```

**示例**:

- 有 instance: `http://localhost:9091/metrics/job/node/instance/192.168.1.100`
- 无 instance: `http://localhost:9091/metrics/job/node`

### 推送的指标来源

node_exporter 默认采集以下指标：

| 指标类别 | 指标名称 | 说明 |
|----------|----------|------|
| CPU | node_cpu_seconds_total | CPU 时间统计 |
| 内存 | node_memory_MemAvailable_bytes | 可用内存 |
| 磁盘 | node_disk_io_time_seconds_total | 磁盘 I/O 时间 |
| 网络 | node_network_receive_bytes_total | 网络接收字节 |
| 文件系统 | node_filesystem_avail_bytes | 文件系统可用空间 |
| 负载 | node_load1 / node_load5 / node_load15 | 系统负载 |

## 命令行参数

### 常用参数

```bash
node-push-exporter --config /path/to/config.yml
```

### 标志 (Flags)

| 标志 | 说明 | 默认值 |
|------|------|--------|
| --config | 配置文件路径 | ./config.yml |
| --version | 显示版本信息 | false |

### 版本信息

```bash
$ node-push-exporter --version

node-push-exporter version v1.0.0 (构建时间: 2024-01-15)
使用 node_exporter 采集指标
```

## 故障排查接口

### 健康检查

由于 node-push-exporter 不直接提供 HTTP API，建议通过以下方式检查：

1. **systemd 服务状态**:
   ```bash
   systemctl status node-push-exporter
   ```

2. **日志检查**:
   ```bash
   journalctl -u node-push-exporter | grep "指标推送成功"
   ```

3. **node_exporter 端点**:
   ```bash
   curl http://localhost:9100/metrics
   ```

4. **Pushgateway 状态**:
   ```bash
   curl http://<pushgateway-url>/metrics | grep "job=\"<your-job>\""
   ```

## 附录: 完整的 systemd 服务定义

```ini
[Unit]
Description=Node Metrics Push Exporter
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/node-push-exporter --config /etc/node-push-exporter/config.yml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
Environment=PATH=/usr/local/bin:/usr/bin:/bin
SyslogIdentifier=node-push-exporter
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true

[Install]
WantedBy=multi-user.target
```