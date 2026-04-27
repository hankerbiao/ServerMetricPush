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

---

## 🐍 Python API 调用

使用 `prometheus-api-client` 库查询 Prometheus。

### 安装

```bash
pip install prometheus-api-client
```

### 基本查询

```python
from prometheus_api_client import PrometheusConnect

prom = PrometheusConnect(url="http://localhost:9090", disable_ssl=True)

# 查询 CPU 使用率
cpu_query = '''
(1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
'''
data = prom.custom_query(query=cpu_query)

for metric in data:
    instance = metric["metric"]["instance"]
    value = float(metric["value"][1])
    print(f"{instance}: {value:.2f}%")
```

### 查询结果格式

```python
[
    {
        "metric": {"instance": "192.168.1.100", "job": "node"},
        "value": [1704067200.0, "23.45"]
    }
]
```

### 查询 GPU 指标

```python
# GPU 利用率
gpu_util = prom.custom_query('gpu_utilization_percent{job="node"}')

# GPU 温度
gpu_temp = prom.custom_query('gpu_temperature_celsius{job="node"}')

# 打印结果
for m in gpu_util:
    print(f"GPU {m['metric']['gpu']}: {m['value'][1]}%")
```

### 批量查询

```python
import concurrent.futures

queries = {
    "cpu": '(1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100',
    "memory": '(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100',
}

with concurrent.futures.ThreadPoolExecutor(max_workers=3) as executor:
    futures = {executor.submit(prom.custom_query, q): k for k, q in queries.items()}
    for future in concurrent.futures.as_completed(futures):
        name = futures[future]
        results = future.result()
        print(f"\n{name}: {len(results)} results")
```