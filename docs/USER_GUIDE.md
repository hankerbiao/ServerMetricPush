# node-push-exporter 用户指南

本文档面向运维人员，提供 node-push-exporter 的详细使用说明。

## 功能概述

node-push-exporter 是一个指标推送服务，其工作流程如下：

```
┌─────────────────┐     /metrics      ┌──────────────────┐
│  node_exporter  │ ◄─── 抓取 ─────── │ node-push-exporter│
│  (子进程)       │                   │                  │
└─────────────────┘                   └────────┬─────────┘
                                               │
                                               │ Push
                                               ▼
                                      ┌──────────────────┐
                                      │ Pushgateway      │
                                      │ (Prometheus)     │
                                      └──────────────────┘
```

**核心功能**：
1. 自动启动和管理 node_exporter 子进程
2. 定时抓取系统指标（CPU、内存、磁盘、网络等）
3. 将指标推送到 Prometheus Pushgateway
4. (可选) 向控制面注册并发送心跳

## 配置参数详解

### pushgateway.* 配置组

#### pushgateway.url
- **说明**: Prometheus Pushgateway 服务地址
- **必填**: 是
- **示例**: `http://10.17.154.252:9091`

#### pushgateway.job
- **说明**: 推送到 Pushgateway 时的 job 标签值
- **必填**: 是
- **示例**: `node`, `production`, `monitoring`

#### pushgateway.instance
- **说明**: 推送到 Pushgateway 时的 instance 标签值
- **必填**: 否
- **留空时**: 自动使用本机 IP 地址
- **示例**: `web-server-01`, `192.168.1.100`

#### pushgateway.interval
- **说明**: 两次推送之间的间隔时间
- **必填**: 是
- **单位**: 秒
- **建议值**: 30-300 秒
- **示例**: `60`

#### pushgateway.timeout
- **说明**: 推送操作的超时时间
- **必填**: 是
- **单位**: 秒
- **建议值**: 10-30 秒
- **示例**: `10`

### node_exporter.* 配置组

#### node_exporter.path
- **说明**: node_exporter 可执行文件的路径
- **必填**: 是
- **查找顺序**:
  1. 绝对/相对路径
  2. 当前目录
  3. 系统 PATH
  4. /usr/local/bin/node_exporter
  5. /usr/bin/node_exporter
  6. /opt/node_exporter/node_exporter
- **示例**: `node_exporter`

#### node_exporter.port
- **说明**: node_exporter 监听的端口
- **必填**: 是
- **默认值**: 9100
- **示例**: `9100`

#### node_exporter.metrics_url
- **说明**: 抓取指标的完整 URL
- **必填**: 是
- **示例**: `http://localhost:9100/metrics`

### control_plane.* 配置组 (可选)

#### control_plane.url
- **说明**: 控制面服务地址
- **必填**: 否（但与 heartbeat_interval 联动）
- **示例**: `http://10.17.154.252:8888`

#### control_plane.heartbeat_interval
- **说明**: 发送心跳的间隔时间
- **必填**: 否（但与 url 联动）
- **单位**: 秒
- **示例**: `30`

## 运维命令

### 服务管理

```bash
# 启动服务
systemctl start node-push-exporter

# 停止服务
systemctl stop node-push-exporter

# 重启服务
systemctl restart node-push-exporter

# 查看服务状态
systemctl status node-push-exporter

# 设置开机自启
systemctl enable node-push-exporter

# 取消开机自启
systemctl disable node-push-exporter
```

### 日志查看

```bash
# 实时查看日志
journalctl -u node-push-exporter -f

# 查看最近 100 行日志
journalctl -u node-push-exporter -n 100

# 查看今天的日志
journalctl -u node-push-exporter --since today

# 查看指定时间的日志
journalctl -u node-push-exporter --since "2024-01-01 10:00:00" --until "2024-01-01 11:00:00"
```

### 指标验证

```bash
# 检查 node_exporter 是否正常运行
curl -s http://127.0.0.1:9100/metrics | grep "^node_cpu" | head -5

# 检查推送是否成功 (查看日志)
journalctl -u node-push-exporter | grep "指标推送成功"

# 检查 Pushgateway 接收状态
curl -s http://<pushgateway-url>/metrics | grep "job=\"<your-job-name>\""
```

## 日志分析

### 正常日志示例

```
启动阶段：
2024/01/15 10:00:00 启动 node-push-exporter 版本 v1.0.0
2024/01/15 10:00:00 Pushgateway地址: http://10.17.154.252:9091
2024/01/15 10:00:00 任务名称: node, 推送间隔: 60秒
2024/01/15 10:00:00 node_exporter 已启动，地址: http://localhost:9100/metrics

运行阶段：
2024/01/15 10:01:00 指标推送成功，来源: http://localhost:9100/metrics
2024/01/15 10:02:00 指标推送成功，来源: http://localhost:9100/metrics
```

### 错误日志分析

| 错误信息 | 可能原因 | 解决方案 |
|----------|----------|----------|
| 加载配置失败 | 配置文件不存在或格式错误 | 检查 /etc/node-push-exporter/config.yml |
| 启动 node_exporter 失败 | node_exporter 二进制不存在或无执行权限 | 检查 /usr/local/bin/node_exporter |
| 获取指标失败 | node_exporter 未正常运行或端口被占用 | 检查 9100 端口 |
| 推送失败 | Pushgateway 服务不可达 | 检查网络和 Pushgateway 服务 |
| 控制面注册失败 | 控制面服务不可达 | 检查 control_plane.url 配置 |

## 故障排查

### 问题 1: 服务启动后立即停止

**排查步骤**:
```bash
# 1. 查看详细错误信息
journalctl -u node-push-exporter -n 50 --no-pager

# 2. 检查配置文件是否存在
ls -la /etc/node-push-exporter/config.yml

# 3. 手动运行测试
/usr/local/bin/node-push-exporter --config /etc/node-push-exporter/config.yml
```

### 问题 2: 指标推送失败

**排查步骤**:
```bash
# 1. 确认 node_exporter 可访问
curl http://127.0.0.1:9100/metrics | head

# 2. 确认 Pushgateway 可访问
curl http://<pushgateway-host>:9091/-/healthy

# 3. 检查防火墙
telnet <pushgateway-host> 9091
```

### 问题 3: 控制面心跳失败

**排查步骤**:
```bash
# 1. 确认控制面服务可访问
curl http://<control-plane-host>:8888/health

# 2. 检查心跳配置
grep control_plane /etc/node-push-exporter/config.yml

# 3. 查看心跳相关日志
journalctl -u node-push-exporter | grep -E "控制面|heartbeat"
```

## 性能调优

### 推送间隔

- **高频监控** (30s): 适用于需要实时监控的场景
- **标准配置** (60s): 推荐默认值，平衡实时性和资源消耗
- **低频推送** (120s+): 适用于大量节点的低负载场景

### 超时设置

- **网络质量好**: 10 秒足够
- **网络延迟高**: 建议 20-30 秒
- **注意**: 超时时间应小于推送间隔

### node_exporter 采集项

默认采集的指标：
- cpu - CPU 使用率
- meminfo - 内存信息
- diskstats - 磁盘统计
- netdev - 网络设备
- filesystem - 文件系统
- loadavg - 负载平均值
- stat - 系统统计
- time - 时间
- uname - 系统信息

## 监控建议

### 监控 node-push-exporter 自身

1. **检查日志中 "指标推送成功" 频率**
2. **监控 Pushgateway 中的指标是否存在**
3. **设置 Pushgateway 告警**: 超过推送间隔 2-3 倍时间无新指标

### 推荐的 Prometheus 告警规则

```yaml
- alert: NodeExporterDown
  expr: up{job="node"} == 0
  for: 2m
  labels:
    severity: critical

- alert: NodeMetricsPushFailed
  expr: rate(pushgateway_push_failures_total[5m]) > 0
  for: 1m
  labels:
    severity: warning
```