---
title: Prometheus 查询示例
description: 常用 PromQL 查询语句和 Python API 调用示例
---

# Prometheus 查询示例

> 💡 本文档提供常用的 PromQL 查询示例，帮助你从 Pushgateway 获取有价值的指标数据。

## 🏷️ 标签说明

所有指标都带有以下标签：

| 标签 | 说明 | 示例 |
|------|------|------|
| `job` | Pushgateway job 名 | `node` |
| `instance` | 实例标识 (IP) | `10.17.154.252` |
| `vendor` | GPU 厂商 | `nvidia`, `rocm` |
| `gpu` | GPU 设备编号 | `0`, `1` |

### 基础查询

```promql
# 获取所有指标
{job="node"}

# 按实例过滤
{job="node", instance="10.17.154.252"}
```

---

## 📊 可用指标清单

### 🖥️ node_exporter 指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `node_cpu_seconds_total` | Counter | CPU 时间（按 mode 分） |
| `node_load1/5/15` | Gauge | 系统负载 |
| `node_memory_*_bytes` | Gauge | 内存相关指标 |
| `node_filesystem_*_bytes` | Gauge | 文件系统容量 |
| `node_disk_*_total` | Counter | 磁盘 I/O |
| `node_network_*_total` | Counter | 网络流量/包数 |
| `node_boot_time_seconds` | Gauge | 启动时间 |
| `node_procs_running` | Gauge | 运行进程数 |

### 🎮 GPU 指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `gpu_info` | Info | GPU 设备信息 |
| `gpu_temperature_celsius` | Gauge | 温度 (°C) |
| `gpu_utilization_percent` | Gauge | 计算利用率 (%) |
| `gpu_memory_*_bytes` | Gauge | 显存使用量 |
| `gpu_memory_used_percent` | Gauge | 显存使用率 (%) |
| `gpu_power_draw_watts` | Gauge | 当前功耗 (W) |
| `gpu_clock_*_hertz` | Gauge | GPU/显存频率 |
| `gpu_fan_speed_percent` | Gauge | 风扇转速 (%) |

### 🔧 自定义指标

| 指标 | 类型 | 说明 |
|------|:----:|------|
| `node_push_exporter_up` | Gauge | 服务运行状态 |
| `node_push_exporter_push_total` | Counter | 推送总次数 |
| `node_push_exporter_push_success_total` | Counter | 推送成功次数 |
| `node_push_exporter_push_failure_total` | Counter | 推送失败次数 |

### 🔍 查看所有指标

```promql
# 列出所有指标名
count by (__name__) ({job="node"})

# 查看特定前缀
{job="node", __name__=~"node_.*"}
{job="node", __name__=~"gpu_.*"}
```

---

## 🖥️ CPU 指标

```promql
# 总 CPU 使用率 (%)
(1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100

# 每核 CPU 使用率
rate(node_cpu_seconds_total{job="node", mode="idle"}[5m])

# 1/5/15 分钟负载
node_load1{job="node"}
node_load5{job="node"}
node_load15{job="node"}
```

---

## 💾 内存指标

```promql
# 内存使用率 (%)
100 - (node_memory_MemAvailable_bytes{job="node"} / node_memory_MemTotal_bytes{job="node"} * 100)

# 内存总量 (GB)
node_memory_MemTotal_bytes{job="node"} / 1024^3

# 可用内存 (GB)
node_memory_MemAvailable_bytes{job="node"} / 1024^3
```

---

## 💿 磁盘指标

```promql
# 磁盘使用率 (%)
(node_filesystem_size_bytes - node_filesystem_avail_bytes) / node_filesystem_size_bytes * 100

# 磁盘读取速率 (MB/s)
rate(node_disk_read_bytes_total{job="node"}[5m]) / 1024 / 1024

# 磁盘写入速率 (MB/s)
rate(node_disk_written_bytes_total{job="node"}[5m]) / 1024 / 1024
```

---

## 🌐 网络指标

```promql
# 接收速率 (MB/s)
rate(node_network_receive_bytes_total{job="node"}[5m]) / 1024 / 1024

# 发送速率 (MB/s)
rate(node_network_transmit_bytes_total{job="node"}[5m]) / 1024 / 1024

# 网络错误率
rate(node_network_receive_errs_total{job="node"}[5m])
rate(node_network_transmit_errs_total{job="node"}[5m])
```

---

## 🌡️ GPU 温度

```promql
# 所有 GPU 温度
gpu_temperature_celsius{job="node"}

# 平均温度 (按厂商)
avg by (vendor) (gpu_temperature_celsius{job="node"})

# 温度 > 80°C 的 GPU
gpu_temperature_celsius{job="node"} > 80
```

---

## 📈 GPU 利用率

```promql
# GPU 利用率 (%)
gpu_utilization_percent{job="node"}

# 按实例统计平均利用率
avg by (instance) (gpu_utilization_percent{job="node"})

# 利用率 > 90% 的 GPU
gpu_utilization_percent{job="node"} > 90
```

---

## 🎮 GPU 显存

```promql
# 显存使用率 (%)
gpu_memory_used_percent{job="node"}

# 显存使用量 (GB)
gpu_memory_used_bytes{job="node"} / 1024^3

# 显存总量 (GB)
gpu_memory_total_bytes{job="node"} / 1024^3

# 使用率 > 80% 的 GPU
gpu_memory_used_percent{job="node"} > 80
```

---

## ⚡ GPU 功耗

```promql
# 单卡功耗 (W)
gpu_power_draw_watts{job="node"}

# 机器总功耗
sum by (instance) (gpu_power_draw_watts{job="node"})

# 单卡平均功耗
avg by (instance) (gpu_power_draw_watts{job="node"})
```

---

## 🔔 告警规则示例

### CPU 使用率过高

```yaml
- alert: HighCPUUsage
  expr: (1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100 > 80
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "CPU 使用率过高 {{ $labels.instance }}"
    description: "CPU 使用率已超过 80%"
```

### GPU 温度过高

```yaml
- alert: GPUTemperatureHigh
  expr: gpu_temperature_celsius{job="node"} > 80
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "GPU {{ $labels.gpu }} 温度过高"
    description: "{{ $labels.instance }} 温度 {{ $value }}°C"
```

### GPU 采集失败

```yaml
- alert: GPUScrapeFailure
  expr: node_push_exporter_gpu_scrape_success{job="node"} == 0
  for: 1m
  labels:
    severity: warning
  annotations:
    summary: "GPU 指标采集失败"
    description: "{{ $labels.instance }} - {{ $labels.vendor }}"
```

### Python API

详细 Python 查询示例请查看 [Python API 查询文档](./python)。

---

## 🐍 Python 批量查询示例

使用 Prometheus HTTP API 进行批量查询，过滤离线节点：

```python
import requests

PROMETHEUS_URL = "http://10.17.154.252:9090"
ALIVE_THRESHOLD = 120  # 节点超时时间（秒）

queries = {
    "cpu_usage_percent": f'clamp_max(avg by (instance) (rate(node_cpu_seconds_total{{mode!~"idle|iowait"}}[5m])) * 100, 100) and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
    "memory_usage_percent": f'(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
    "network_throughput_mbps": f'sum by (instance) (rate(node_network_receive_bytes_total{{device!~"lo|docker.*|veth.*"}}[5m]) + rate(node_network_transmit_bytes_total{{device!~"lo|docker.*|veth.*"}}[5m])) / 1024 / 1024 and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
    "disk_io_mbps": f'sum by (instance) (rate(node_disk_read_bytes_total{{device!~"loop.*|ram.*"}}[5m]) + rate(node_disk_written_bytes_total{{device!~"loop.*|ram.*"}}[5m])) / 1024 / 1024 and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
    "gpu_utilization_percent": f'avg by (instance) (gpu_utilization_percent{{job="node"}}) and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
    "gpu_memory_usage_percent": f'avg by (instance) (gpu_memory_used_percent{{job="node"}}) and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
    "gpu_memory_used_gb": f'sum by (instance) (gpu_memory_used_bytes{{job="node"}}) / 1024 / 1024 / 1024 and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
    "gpu_temperature_celsius": f'avg by (instance) (gpu_temperature_celsius{{job="node"}}) and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
    "gpu_power_draw_watts": f'sum by (instance) (gpu_power_draw_watts{{job="node"}}) and on(instance) (time() - push_time_seconds < {ALIVE_THRESHOLD})',
}

def query_prometheus(query: str) -> list:
    """执行 Prometheus 查询"""
    response = requests.post(
        f"{PROMETHEUS_URL}/api/v1/query",
        data={"query": query}
    )
    return response.json().get("data", {}).get("result", [])

def get_all_metrics() -> dict:
    """获取所有指标"""
    results = {}
    for name, query in queries.items():
        data = query_prometheus(query)
        results[name] = {
            m["metric"]["instance"]: float(m["value"][1])
            for m in data
        }
    return results

# 使用示例
metrics = get_all_metrics()
for name, values in metrics.items():
    print(f"\n{name}:")
    for instance, value in values.items():
        print(f"  {instance}: {value:.2f}")
```

```