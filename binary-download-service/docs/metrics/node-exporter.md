---
title: Node Exporter 指标
description: node_exporter 标准系统指标详解
---

# Node Exporter 指标

Node Push Exporter 内部运行 node_exporter 来采集标准系统指标。

## 🖥️ CPU 指标

### `node_cpu_seconds_total`

CPU 时间累计值（秒），按 CPU 核心和模式统计。

| 属性 | 值 |
|------|-----|
| 类型 | Counter |
| 标签 | `cpu`, `mode` |
| 模式 | `idle`, `user`, `system`, `iowait`, `irq`, `softirq`, `steal` |

```promql
# 每核每秒 CPU 使用时间
rate(node_cpu_seconds_total{job="node"}[5m])

# 计算 CPU 使用率
(1 - rate(node_cpu_seconds_total{mode="idle"}[5m]) / 4) * 100
```

### `node_load1` / `node_load5` / `node_load15`

系统平均负载（1/5/15分钟）。

| 属性 | 值 |
|------|-----|
| 类型 | Gauge |

---

## 💾 内存指标

### 核心指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `node_memory_MemTotal_bytes` | Gauge | 总物理内存 |
| `node_memory_MemFree_bytes` | Gauge | 空闲内存 |
| `node_memory_MemAvailable_bytes` | Gauge | 可用内存 (含 buffers/cached) |
| `node_memory_Buffers_bytes` | Gauge | Buffers 内存 |
| `node_memory_Cached_bytes` | Gauge | Cached 内存 |

```promql
# 内存使用率 (%)
(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100
```

### Swap 指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `node_memory_SwapTotal_bytes` | Gauge | Swap 总量 |
| `node_memory_SwapFree_bytes` | Gauge | Swap 空闲量 |

```promql
# Swap 使用率 (%)
(node_memory_SwapTotal_bytes - node_memory_SwapFree_bytes) / node_memory_SwapTotal_bytes * 100
```

---

## 💿 磁盘 / 文件系统指标

### 文件系统容量

| 指标 | 类型 | 标签 | 说明 |
|------|:----:|------|------|
| `node_filesystem_size_bytes` | Gauge | `device`, `fstype`, `mountpoint` | 总大小 |
| `node_filesystem_avail_bytes` | Gauge | `device`, `fstype`, `mountpoint` | 可用空间 |

```promql
# 磁盘使用率 (%)
(node_filesystem_size_bytes - node_filesystem_avail_bytes) / node_filesystem_size_bytes * 100
```

### 磁盘 I/O

| 指标 | 类型 | 标签 | 说明 |
|------|:----:|------|------|
| `node_disk_read_bytes_total` | Counter | `device` | 读取字节数 |
| `node_disk_written_bytes_total` | Counter | `device` | 写入字节数 |
| `node_disk_reads_completed_total` | Counter | `device` | 读取次数 |
| `node_disk_writes_completed_total` | Counter | `device` | 写入次数 |
| `node_disk_io_time_seconds_total` | Counter | `device` | I/O 耗时 |

```promql
# 读取速率 (MB/s)
rate(node_disk_read_bytes_total{job="node"}[5m]) / 1024 / 1024

# 写入速率 (MB/s)
rate(node_disk_written_bytes_total{job="node"}[5m]) / 1024 / 1024
```

---

## 🌐 网络指标

| 指标 | 类型 | 标签 | 说明 |
|------|:----:|------|------|
| `node_network_receive_bytes_total` | Counter | `device` | 接收字节数 |
| `node_network_transmit_bytes_total` | Counter | `device` | 发送字节数 |
| `node_network_receive_packets_total` | Counter | `device` | 接收数据包数 |
| `node_network_transmit_packets_total` | Counter | `device` | 发送数据包数 |
| `node_network_receive_errs_total` | Counter | `device` | 接收错误数 |
| `node_network_transmit_errs_total` | Counter | `device` | 发送错误数 |

```promql
# 接收速率 (MB/s)
rate(node_network_receive_bytes_total{job="node"}[5m]) / 1024 / 1024

# 发送速率 (MB/s)
rate(node_network_transmit_bytes_total{job="node"}[5m]) / 1024 / 1024

# 网络错误率
rate(node_network_receive_errs_total{job="node"}[5m])
```

---

## ⏰ 系统指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `node_time_seconds` | Gauge | 当前系统时间 (Unix timestamp) |
| `node_boot_time_seconds` | Gauge | 系统启动时间 |
| `node_uptime_seconds` | Gauge | 系统运行时长 (秒) |

```promql
# 系统运行时长 (天)
node_uptime_seconds{job="node"} / 86400
```

---

## 📊 进程指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `node_procs_running` | Gauge | 运行中的进程数 |
| `node_procs_blocked` | Gauge | 阻塞的进程数 |

---

## 📁 文件句柄指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `node_filefd_allocated` | Gauge | 已分配的文件句柄 |
| `node_filefd_maximum` | Gauge | 文件句柄最大值 |

```promql
# 文件句柄使用率 (%)
node_filefd_allocated{job="node"} / node_filefd_maximum{job="node"} * 100
```

---

## 🔍 查看所有指标

```bash
# 本地查看
curl http://localhost:9100/metrics | grep "^node_"

# 在 Prometheus 中查看
{job="node", __name__=~"node_.*"}
```